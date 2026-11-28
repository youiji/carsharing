package services

import (
	"context"
	"errors"
	"slices"
	"strings"

	"carsharing/internal/dto"
	"carsharing/internal/models"
	"carsharing/internal/repository"
	"carsharing/internal/util"
)

type CarService struct {
	cars *repository.CarRepository
}

func NewCarService(cars *repository.CarRepository) *CarService {
	return &CarService{cars: cars}
}

func (s *CarService) List(ctx context.Context, user models.User, f models.CarFilter) (dto.List[dto.Car], error) {
	if user.Role == models.RoleClient {
		available := models.CarAvailable
		f.Status = &available
	}
	cars, err := s.cars.List(ctx, f)
	if err != nil {
		return dto.List[dto.Car]{}, err
	}
	return dto.NewList(dto.NewCars(cars)), nil
}

var errCarNotFound = util.ErrNotFound.WithMessage("Автомобиль не найден")

func (s *CarService) Get(ctx context.Context, user models.User, id int64) (dto.Car, error) {
	car, err := s.cars.GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return dto.Car{}, errCarNotFound
	}
	if err != nil {
		return dto.Car{}, err
	}
	if user.Role == models.RoleClient && car.Status != models.CarAvailable {
		own, err := s.cars.InUserOpenRental(ctx, id, user.ID)
		if err != nil {
			return dto.Car{}, err
		}
		if !own {
			return dto.Car{}, errCarNotFound
		}
	}
	return dto.NewCar(car), nil
}

var manualCarTransitions = map[string][]string{
	models.CarAvailable:      {models.CarMaintenance, models.CarDecommissioned},
	models.CarMaintenance:    {models.CarAvailable, models.CarDecommissioned},
	models.CarDecommissioned: {models.CarAvailable},
}

var ManualCarStatuses = []string{models.CarAvailable, models.CarMaintenance, models.CarDecommissioned}

var carStatusNames = map[string]string{
	models.CarAvailable:      "свободен",
	models.CarBooked:         "забронирован",
	models.CarInUse:          "в поездке",
	models.CarMaintenance:    "на обслуживании",
	models.CarDecommissioned: "списан",
}

func (s *CarService) Create(ctx context.Context, req dto.CarRequest) (dto.Car, error) {
	car, err := validateCar(req)
	if err != nil {
		return dto.Car{}, err
	}
	id, err := s.cars.Create(ctx, car)
	if err != nil {
		return dto.Car{}, err
	}
	return s.get(ctx, id)
}

func (s *CarService) Update(ctx context.Context, id int64, req dto.CarRequest) (dto.Car, error) {
	car, err := validateCar(req)
	if err != nil {
		return dto.Car{}, err
	}
	car.ID = id
	if err := s.cars.Update(ctx, car); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return dto.Car{}, errCarNotFound
		}
		return dto.Car{}, err
	}
	return s.get(ctx, id)
}

func (s *CarService) ChangeStatus(ctx context.Context, id int64, req dto.CarStatusRequest) (dto.Car, error) {
	to := strings.TrimSpace(req.Status)
	if !slices.Contains(ManualCarStatuses, to) {
		return dto.Car{}, util.ErrValidation.WithDetails(map[string]string{
			"status": "допустимые значения: " + strings.Join(ManualCarStatuses, ", "),
		})
	}
	err := s.cars.ChangeStatus(ctx, id, to, func(from string) error {
		if from == models.CarBooked || from == models.CarInUse {
			return util.ErrInvalidTransition.WithMessage(
				"Автомобиль " + carStatusNames[from] + ": статус меняется только через аренду")
		}
		if !slices.Contains(manualCarTransitions[from], to) {
			return util.ErrInvalidTransition.WithMessage(
				"Нельзя перевести автомобиль из статуса «" + carStatusNames[from] + "» в «" + carStatusNames[to] + "»")
		}
		return nil
	})
	if errors.Is(err, repository.ErrNotFound) {
		return dto.Car{}, errCarNotFound
	}
	if err != nil {
		return dto.Car{}, err
	}
	return s.get(ctx, id)
}

func (s *CarService) Delete(ctx context.Context, id int64) error {
	err := s.cars.Delete(ctx, id)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return errCarNotFound
	case errors.Is(err, util.ErrHasRelatedRecords):
		return util.ErrHasRelatedRecords.WithMessage(
			"Нельзя удалить автомобиль: у него есть аренды или записи обслуживания. Переведите его в статус «списан»")
	}
	return err
}

func (s *CarService) get(ctx context.Context, id int64) (dto.Car, error) {
	car, err := s.cars.GetByID(ctx, id)
	if err != nil {
		return dto.Car{}, err
	}
	return dto.NewCar(car), nil
}

func validateCar(req dto.CarRequest) (models.Car, error) {
	var v util.Validator
	brand := strings.TrimSpace(req.BrandModel)
	plate := util.NormalizePlate(req.PlateNumber)

	switch {
	case req.TariffID == nil:
		v.Add("tariff_id", "обязательное поле")
	case *req.TariffID <= 0:
		v.Add("tariff_id", "должно быть положительным целым числом")
	}
	v.StringLen("brand_model", brand, 1, 100)
	v.Check(util.ValidPlate(plate), "plate_number",
		"формат А123ВС77 или А123ВС777, допустимые буквы: А В Е К М Н О Р С Т У Х")
	switch {
	case req.FuelLevel == nil:
		v.Add("fuel_level", "обязательное поле")
	case *req.FuelLevel < 0 || *req.FuelLevel > 100:
		v.Add("fuel_level", "должно быть от 0 до 100")
	}
	switch {
	case req.Latitude == nil:
		v.Add("latitude", "обязательное поле")
	case *req.Latitude < -90 || *req.Latitude > 90:
		v.Add("latitude", "должно быть от -90 до 90")
	}
	switch {
	case req.Longitude == nil:
		v.Add("longitude", "обязательное поле")
	case *req.Longitude < -180 || *req.Longitude > 180:
		v.Add("longitude", "должно быть от -180 до 180")
	}
	if err := v.Err(); err != nil {
		return models.Car{}, err
	}
	return models.Car{
		TariffID:    *req.TariffID,
		BrandModel:  brand,
		PlateNumber: plate,
		FuelLevel:   *req.FuelLevel,
		Latitude:    *req.Latitude,
		Longitude:   *req.Longitude,
	}, nil
}
