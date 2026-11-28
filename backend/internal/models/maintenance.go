package models

import (
	"time"

	"carsharing/internal/util"
)

// MaintenanceLog — строка maintenance_logs. Car* и ManagerFullName приходят из JOIN.
type MaintenanceLog struct {
	ID          int64
	CarID       int64
	ManagerID   int64
	Description string
	Cost        util.Money
	LoggedAt    time.Time

	CarBrandModel   string
	CarPlateNumber  string
	ManagerFullName string
}

type MaintenanceFilter struct {
	CarID  *int64
	Search string
	Sort   string
	Desc   bool
}
