package services

import (
	"context"
	"errors"
	"time"

	"carsharing/internal/dto"
	"carsharing/internal/models"
	"carsharing/internal/repository"
	"carsharing/internal/util"
)

type RentalService struct {
	repo *repository.RentalRepository
}

func NewRentalService(repo *repository.RentalRepository) *RentalService {
	return &RentalService{repo: repo}
}

func now() time.Time {
	return time.Now().Truncate(time.Microsecond)
}

func (s *RentalService) Book(ctx context.Context, user models.User, req dto.CreateRentalRequest) (dto.Rental, error) {
	if req.CarID <= 0 {
		return dto.Rental{}, util.ErrValidation.WithDetails(map[string]string{"car_id": "обязательное поле, положительное целое"})
	}
	if user.DriverLicense == nil || *user.DriverLicense == "" {
		return dto.Rental{}, util.ErrLicenseRequired
	}
	if _, err := s.ExpireBookings(ctx); err != nil {
		return dto.Rental{}, err
	}

	var rentalID int64
	err := s.repo.InTx(ctx, func(tx *repository.RentalTx) error {
		car, err := tx.LockCar(ctx, req.CarID)
		if errors.Is(err, repository.ErrNotFound) {
			return util.ErrNotFound.WithMessage("Автомобиль не найден")
		}
		if err != nil {
			return err
		}
		if car.Status != models.CarAvailable {
			return util.ErrCarUnavailable
		}

		role, license, err := tx.LockUser(ctx, user.ID)
		if errors.Is(err, repository.ErrNotFound) {
			return util.ErrUnauthorized
		}
		if err != nil {
			return err
		}
		if role != models.RoleClient {
			return util.ErrForbidden
		}
		if license == nil || *license == "" {
			return util.ErrLicenseRequired
		}

		hasOpen, err := tx.UserHasOpenRental(ctx, user.ID)
		if err != nil {
			return err
		}
		if hasOpen {
			return util.ErrUserHasActiveRental
		}

		rentalID, err = tx.Insert(ctx, models.Rental{
			UserID:        user.ID,
			TariffID:      car.TariffID,
			CarID:         car.ID,
			BookedAt:      now(),
			PricePerMin:   car.PricePerMin,
			WaitingPerMin: car.WaitingPerMin,
		})
		if err != nil {
			return err
		}
		return tx.SetCarStatus(ctx, car.ID, models.CarAvailable, models.CarBooked)
	})
	if err != nil {
		return dto.Rental{}, err
	}

	rental, err := s.repo.GetByID(ctx, rentalID)
	if err != nil {
		return dto.Rental{}, err
	}
	return dto.NewRental(rental), nil
}

func CalcRentalCost(bookedAt, startedAt, endedAt time.Time, pricePerMin, waitingPerMin util.Money) util.Money {
	rideMin := max(1, ceilMinutes(endedAt.Sub(startedAt)))
	waitMin := ceilMinutes(startedAt.Sub(bookedAt))
	return pricePerMin*util.Money(rideMin) + waitingPerMin*util.Money(waitMin)
}

func CancellationFee(bookedAt, cancelledAt time.Time, waitingPerMin util.Money) util.Money {
	minutes := min(ceilMinutes(cancelledAt.Sub(bookedAt)), int64(models.BookingTTL/time.Minute))
	return waitingPerMin * util.Money(minutes)
}

func ceilMinutes(d time.Duration) int64 {
	if d <= 0 {
		return 0
	}
	return int64((d + time.Minute - 1) / time.Minute)
}

func (s *RentalService) Start(ctx context.Context, user models.User, rentalID int64) (dto.Rental, error) {
	expired := false
	err := s.repo.InTx(ctx, func(tx *repository.RentalTx) error {
		rental, err := lockOwned(ctx, tx, user, rentalID, false)
		if err != nil {
			return err
		}
		if rental.Status != models.RentalBooked {
			return transitionErr(rental.Status, "начать поездку можно только по бронированию")
		}
		if isExpired(rental) {
			expired = true
			return expire(ctx, tx, rental)
		}
		if err := tx.Start(ctx, rental.ID, now()); err != nil {
			return err
		}
		return tx.SetCarStatus(ctx, rental.CarID, models.CarBooked, models.CarInUse)
	})
	if err != nil {
		return dto.Rental{}, err
	}
	if expired {
		return dto.Rental{}, util.ErrBookingExpired
	}
	return s.get(ctx, rentalID)
}

func (s *RentalService) Finish(ctx context.Context, user models.User, rentalID int64, req dto.FinishRentalRequest) (dto.Rental, error) {
	if err := validateFinish(req); err != nil {
		return dto.Rental{}, err
	}
	err := s.repo.InTx(ctx, func(tx *repository.RentalTx) error {
		rental, err := lockOwned(ctx, tx, user, rentalID, false)
		if err != nil {
			return err
		}
		if rental.Status != models.RentalActive || rental.StartedAt == nil {
			return transitionErr(rental.Status, "завершить можно только начатую поездку")
		}
		endedAt := now()
		cost := CalcRentalCost(rental.BookedAt, *rental.StartedAt, endedAt, rental.PricePerMin, rental.WaitingPerMin)
		if err := tx.Complete(ctx, rental.ID, endedAt, cost); err != nil {
			return err
		}
		return tx.ReleaseCar(ctx, rental.CarID, models.CarLocation{Latitude: req.Latitude, Longitude: req.Longitude})
	})
	if err != nil {
		return dto.Rental{}, err
	}
	return s.get(ctx, rentalID)
}

func (s *RentalService) Cancel(ctx context.Context, user models.User, rentalID int64) (dto.Rental, error) {
	err := s.repo.InTx(ctx, func(tx *repository.RentalTx) error {
		rental, err := lockOwned(ctx, tx, user, rentalID, isStaff(user))

		if isExpired(rental) {
			return expire(ctx, tx, rental)
		}
		if err != nil {
			return err
		}
		if rental.Status != models.RentalBooked {
			return transitionErr(rental.Status, "отменить можно только бронирование")
		}
		cancelledAt := now()
		fee := util.Money(0)
		if !isStaff(user) {
			fee = CancellationFee(rental.BookedAt, cancelledAt, rental.WaitingPerMin)
		}
		if err := tx.Cancel(ctx, rental.ID, cancelledAt, fee); err != nil {
			return err
		}
		return tx.SetCarStatus(ctx, rental.CarID, models.CarBooked, models.CarAvailable)
	})
	if err != nil {
		return dto.Rental{}, err
	}
	return s.get(ctx, rentalID)
}

func (s *RentalService) get(ctx context.Context, id int64) (dto.Rental, error) {
	rental, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return dto.Rental{}, err
	}
	return dto.NewRental(rental), nil
}

func lockOwned(ctx context.Context, tx *repository.RentalTx, user models.User, id int64, anyOwner bool) (models.Rental, error) {
	rental, err := tx.LockRental(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return models.Rental{}, util.ErrNotFound.WithMessage("Аренда не найдена")
	}
	if err != nil {
		return models.Rental{}, err
	}
	if !anyOwner && rental.UserID != user.ID {
		return models.Rental{}, util.ErrForbidden.WithMessage("Это чужая аренда")
	}
	return rental, nil
}

func isStaff(u models.User) bool {
	return u.Role == models.RoleManager || u.Role == models.RoleAdmin
}

var rentalStatusNames = map[string]string{
	models.RentalBooked:    "забронирована",
	models.RentalActive:    "в поездке",
	models.RentalCompleted: "завершена",
	models.RentalCancelled: "отменена",
}

func transitionErr(status, hint string) error {
	return util.ErrInvalidTransition.WithMessage("Аренда " + rentalStatusNames[status] + ": " + hint)
}

func validateFinish(req dto.FinishRentalRequest) error {
	var v util.Validator
	if (req.Latitude == nil) != (req.Longitude == nil) {
		v.Add("latitude", "latitude и longitude передаются только вместе")
	}
	if req.Latitude != nil {
		v.Check(*req.Latitude >= -90 && *req.Latitude <= 90, "latitude", "должно быть от -90 до 90")
	}
	if req.Longitude != nil {
		v.Check(*req.Longitude >= -180 && *req.Longitude <= 180, "longitude", "должно быть от -180 до 180")
	}
	return v.Err()
}

func (s *RentalService) ExpireBookings(ctx context.Context) (int, error) {
	n := 0
	err := s.repo.InTx(ctx, func(tx *repository.RentalTx) error {
		expired, err := tx.LockExpiredBookings(ctx, now().Add(-models.BookingTTL))
		if err != nil {
			return err
		}
		for _, r := range expired {
			if err := expire(ctx, tx, r); err != nil {
				return err
			}
		}
		n = len(expired)
		return nil
	})
	return n, err
}

func isExpired(r models.Rental) bool {
	return !now().Before(r.BookedAt.Add(models.BookingTTL))
}

func expire(ctx context.Context, tx *repository.RentalTx, r models.Rental) error {
	end := r.BookedAt.Add(models.BookingTTL)
	if err := tx.Cancel(ctx, r.ID, end, CancellationFee(r.BookedAt, end, r.WaitingPerMin)); err != nil {
		return err
	}
	return tx.SetCarStatus(ctx, r.CarID, models.CarBooked, models.CarAvailable)
}

func (s *RentalService) Get(ctx context.Context, user models.User, rentalID int64) (dto.Rental, error) {
	rental, err := s.repo.GetByID(ctx, rentalID)
	if errors.Is(err, repository.ErrNotFound) {
		return dto.Rental{}, util.ErrNotFound.WithMessage("Аренда не найдена")
	}
	if err != nil {
		return dto.Rental{}, err
	}
	if !isStaff(user) && rental.UserID != user.ID {
		return dto.Rental{}, util.ErrForbidden.WithMessage("Это чужая аренда")
	}
	return dto.NewRental(rental), nil
}

func (s *RentalService) ListMy(ctx context.Context, user models.User, f models.RentalFilter) (dto.List[dto.Rental], error) {
	f.UserID = &user.ID
	f.CarID, f.Search = nil, ""
	return s.list(ctx, f)
}

func (s *RentalService) List(ctx context.Context, f models.RentalFilter) (dto.List[dto.Rental], error) {
	return s.list(ctx, f)
}

func (s *RentalService) list(ctx context.Context, f models.RentalFilter) (dto.List[dto.Rental], error) {
	rentals, err := s.repo.List(ctx, f)
	if err != nil {
		return dto.List[dto.Rental]{}, err
	}
	return dto.NewList(dto.NewRentals(rentals)), nil
}
