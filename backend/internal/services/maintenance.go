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

type MaintenanceService struct {
	logs *repository.MaintenanceRepository
}

func NewMaintenanceService(logs *repository.MaintenanceRepository) *MaintenanceService {
	return &MaintenanceService{logs: logs}
}

const maxMaintenanceCost = util.Money(9_999_999_999) // NUMERIC(10,2): 99 999 999.99

var errMaintenanceNotFound = util.ErrNotFound.WithMessage("Запись обслуживания не найдена")

func (s *MaintenanceService) List(ctx context.Context, f models.MaintenanceFilter) (dto.List[dto.Maintenance], error) {
	logs, err := s.logs.List(ctx, f)
	if err != nil {
		return dto.List[dto.Maintenance]{}, err
	}
	out := make([]dto.Maintenance, len(logs))
	for i, m := range logs {
		out[i] = dto.NewMaintenance(m)
	}
	return dto.NewList(out), nil
}

func (s *MaintenanceService) Create(ctx context.Context, author models.User, req dto.CreateMaintenanceRequest) (dto.Maintenance, error) {
	var v util.Validator
	v.Check(req.CarID > 0, "car_id", "обязательное поле, положительное целое")
	description := strings.TrimSpace(req.Description)
	v.StringLen("description", description, 1, 255)
	if req.Cost == nil {
		v.Add("cost", "обязательное поле")
	} else {
		checkCost(&v, *req.Cost)
	}
	if err := v.Err(); err != nil {
		return dto.Maintenance{}, err
	}

	id, err := s.logs.Create(ctx, models.MaintenanceLog{
		CarID:       req.CarID,
		ManagerID:   author.ID,
		Description: description,
		Cost:        *req.Cost,
	})
	if err != nil {
		return dto.Maintenance{}, err
	}
	return s.get(ctx, id)
}

func (s *MaintenanceService) Update(ctx context.Context, user models.User, id int64, req dto.UpdateMaintenanceRequest) (dto.Maintenance, error) {
	var v util.Validator
	if req.Description == nil && req.Cost == nil {
		v.Add("description", "передайте description и/или cost")
	}
	var description *string
	if req.Description != nil {
		d := strings.TrimSpace(*req.Description)
		v.StringLen("description", d, 1, 255)
		description = &d
	}
	if req.Cost != nil {
		checkCost(&v, *req.Cost)
	}
	if err := v.Err(); err != nil {
		return dto.Maintenance{}, err
	}

	if err := s.checkOwner(ctx, user, id); err != nil {
		return dto.Maintenance{}, err
	}
	if err := s.logs.Update(ctx, id, description, req.Cost); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return dto.Maintenance{}, errMaintenanceNotFound
		}
		return dto.Maintenance{}, err
	}
	return s.get(ctx, id)
}

func (s *MaintenanceService) Delete(ctx context.Context, user models.User, id int64) error {
	if err := s.checkOwner(ctx, user, id); err != nil {
		return err
	}
	if err := s.logs.Delete(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return errMaintenanceNotFound
		}
		return err
	}
	return nil
}

// 404 — записи нет; 403 — менеджер трогает чужую.
func (s *MaintenanceService) checkOwner(ctx context.Context, user models.User, id int64) error {
	m, err := s.logs.GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return errMaintenanceNotFound
	}
	if err != nil {
		return err
	}
	if user.Role != models.RoleAdmin && m.ManagerID != user.ID {
		return util.ErrForbidden.WithMessage("Менеджер может изменять только свои записи")
	}
	return nil
}

func (s *MaintenanceService) get(ctx context.Context, id int64) (dto.Maintenance, error) {
	m, err := s.logs.GetByID(ctx, id)
	if err != nil {
		return dto.Maintenance{}, err
	}
	return dto.NewMaintenance(m), nil
}

func checkCost(v *util.Validator, cost util.Money) {
	v.Check(cost >= 0 && cost <= maxMaintenanceCost, "cost", "должно быть от 0 до 99999999.99")
}
