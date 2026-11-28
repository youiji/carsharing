package dto

import (
	"time"

	"carsharing/internal/models"
	"carsharing/internal/util"
)

type MaintenanceCar struct {
	ID          int64  `json:"id"`
	BrandModel  string `json:"brand_model"`
	PlateNumber string `json:"plate_number"`
}

type MaintenanceManager struct {
	ID       int64  `json:"id"`
	FullName string `json:"full_name"`
}

type Maintenance struct {
	ID          int64              `json:"id"`
	Car         MaintenanceCar     `json:"car"`
	Manager     MaintenanceManager `json:"manager"`
	Description string             `json:"description"`
	Cost        util.Money         `json:"cost"`
	LoggedAt    time.Time          `json:"logged_at"`
}

// manager_id в запросе нет намеренно: автор = текущий пользователь.
type CreateMaintenanceRequest struct {
	CarID       int64       `json:"car_id"`
	Description string      `json:"description"`
	Cost        *util.Money `json:"cost"`
}

// nil = поле не прислали, не меняем.
type UpdateMaintenanceRequest struct {
	Description *string     `json:"description"`
	Cost        *util.Money `json:"cost"`
}

func NewMaintenance(m models.MaintenanceLog) Maintenance {
	return Maintenance{
		ID:          m.ID,
		Car:         MaintenanceCar{ID: m.CarID, BrandModel: m.CarBrandModel, PlateNumber: m.CarPlateNumber},
		Manager:     MaintenanceManager{ID: m.ManagerID, FullName: m.ManagerFullName},
		Description: m.Description,
		Cost:        m.Cost,
		LoggedAt:    m.LoggedAt,
	}
}
