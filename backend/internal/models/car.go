package models

import (
	"carsharing/internal/util"
)

const (
	CarAvailable      = "available"
	CarBooked         = "booked"
	CarInUse          = "int_use"
	CarMaintenance    = "maintenance"
	carDecommissioned = "decommissioned"
)

var CarStatuses = []string{
	CarAvailable,
	CarBooked,
	CarInUse,
	CarMaintenance,
	carDecommissioned}

type Tariff struct {
	ID            int64
	Name          string
	PricePerMin   util.Money
	WaitingPerMin util.Money
}

type Car struct {
	ID          int64
	TariffID    int64
	BrandModel  string
	PlateNumber string
	FuelLevel   int
	Latitude    float64
	Longitude   float64
	Status      string
	Tariff      Tariff
}

type CarFilter struct {
	Status   *string
	TariffID *int64
	Search   string
	Sort     string
	Desc     bool
}
