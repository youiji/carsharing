package services

import (
	"context"
	"errors"
	"strings"

	"carsharing/internal/dto"
	"carsharing/internal/models"
	"carsharing/internal/repository"
	"carsharing/internal/util"
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

const maxTariffPrice = util.Money(99999999) // 999 999.99

var errTariffNotFound = util.ErrNotFound.WithMessage("Тариф не найден")

func (s *TariffService) Create(ctx context.Context, req dto.TariffRequest) (dto.Tariff, error) {
	t, err := validateTariff(req)
	if err != nil {
		return dto.Tariff{}, err
	}
	created, err := s.tariffs.Create(ctx, t)
	if err != nil {
		return dto.Tariff{}, err
	}
	return dto.NewTariff(created), nil
}

func (s *TariffService) Update(ctx context.Context, id int64, req dto.TariffRequest) (dto.Tariff, error) {
	t, err := validateTariff(req)
	if err != nil {
		return dto.Tariff{}, err
	}
	t.ID = id
	updated, err := s.tariffs.Update(ctx, t)
	if errors.Is(err, repository.ErrNotFound) {
		return dto.Tariff{}, errTariffNotFound
	}
	if err != nil {
		return dto.Tariff{}, err
	}
	return dto.NewTariff(updated), nil
}

func (s *TariffService) Delete(ctx context.Context, id int64) error {
	err := s.tariffs.Delete(ctx, id)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return errTariffNotFound
	case errors.Is(err, util.ErrHasRelatedRecords):
		return util.ErrHasRelatedRecords.WithMessage("Нельзя удалить тариф: он используется автомобилями или арендами")
	}
	return err
}

func validateTariff(req dto.TariffRequest) (models.Tariff, error) {
	var v util.Validator
	name := strings.TrimSpace(req.Name)
	v.StringLen("name", name, 1, 50)
	checkPrice := func(field string, p *util.Money) {
		switch {
		case p == nil:
			v.Add(field, "обязательное поле")
		case *p < 0 || *p > maxTariffPrice:
			v.Add(field, "должно быть от 0 до 999999.99")
		}
	}
	checkPrice("price_per_min", req.PricePerMin)
	checkPrice("waiting_per_min", req.WaitingPerMin)
	if err := v.Err(); err != nil {
		return models.Tariff{}, err
	}
	return models.Tariff{Name: name, PricePerMin: *req.PricePerMin, WaitingPerMin: *req.WaitingPerMin}, nil
}
