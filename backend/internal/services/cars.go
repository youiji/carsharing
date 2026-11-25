package services

import (
	"context"
	"errors"

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
