package services

import (
	"context"

	"carsharing/internal/dto"
	"carsharing/internal/repository"
)

type TariffService struct {
	tariffs *repository.TariffRepository
}

func NewTariffService(tariffs *repository.TariffRepository) *TariffService {
	return &TariffService{tariffs: tariffs}
}

func (s *TariffService) List(ctx context.Context) (dto.List[dto.Tariff], error) {
	ts, err := s.tariffs.List(ctx)
	if err != nil {
		return dto.List[dto.Tariff]{}, err
	}
	out := make([]dto.Tariff, len(ts))
	for i, t := range ts {
		out[i] = dto.NewTariff(t)
	}
	return dto.NewList(out), nil
}
