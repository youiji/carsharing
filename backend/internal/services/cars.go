package services

import (
	"context"

	"carsharing/internal/dto"
	"carsharing/internal/models"
	"carsharing/internal/repository"
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
