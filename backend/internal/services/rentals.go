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
