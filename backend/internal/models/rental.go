package models

import (
	"time"

	"carsharing/internal/util"
)

const (
	RentalBooked    = "booked"
	RentalActive    = "active"
	RentalCompleted = "completed"
	RentalCancelled = "cancelled"
)

const BookingTTL = 20 * time.Minute

type Rental struct {
	ID            int64
	UserID        int64
	TariffID      int64
	CarID         int64
	BookedAt      time.Time
	StartedAt     *time.Time
	EndedAt       *time.Time
	PricePerMin   util.Money
	WaitingPerMin util.Money
	TotalCost     *util.Money
	Status        string

	UserFullName   string
	CarBrandModel  string
	CarPlateNumber string
}

type CarLocation struct {
	Latitude  *float64
	Longitude *float64
}

var RentalStatuses = []string{RentalBooked, RentalActive, RentalCompleted, RentalCancelled}

type RentalFilter struct {
	UserID *int64
	CarID  *int64
	Status *string
	Search string
	Sort   string
	Desc   bool
}
