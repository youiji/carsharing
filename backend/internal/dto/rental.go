package dto

import (
	"time"

	"carsharing/internal/models"
	"carsharing/internal/util"
)

type CreateRentalRequest struct {
	CarID int64 `json:"car_id"`
}

type RentalUser struct {
	ID       int64  `json:"id"`
	FullName string `json:"full_name"`
}

type RentalCar struct {
	ID          int64  `json:"id"`
	BrandModel  string `json:"brand_model"`
	PlateNumber string `json:"plate_number"`
}

type Rental struct {
	ID            int64       `json:"id"`
	User          RentalUser  `json:"user"`
	Car           RentalCar   `json:"car"`
	TariffID      int64       `json:"tariff_id"`
	BookedAt      time.Time   `json:"booked_at"`
	ExpiresAt     *time.Time  `json:"expires_at"`
	StartedAt     *time.Time  `json:"started_at"`
	EndedAt       *time.Time  `json:"ended_at"`
	PricePerMin   util.Money  `json:"price_per_min"`
	WaitingPerMin util.Money  `json:"waiting_per_min"`
	TotalCost     *util.Money `json:"total_cost"`
	Status        string      `json:"status"`
}

func NewRental(r models.Rental) Rental {
	var expires *time.Time
	if r.Status == models.RentalBooked {
		t := r.BookedAt.Add(models.BookingTTL)
		expires = &t
	}
	return Rental{
		ID:            r.ID,
		User:          RentalUser{ID: r.UserID, FullName: r.UserFullName},
		Car:           RentalCar{ID: r.CarID, BrandModel: r.CarBrandModel, PlateNumber: r.CarPlateNumber},
		TariffID:      r.TariffID,
		BookedAt:      r.BookedAt,
		ExpiresAt:     expires,
		StartedAt:     r.StartedAt,
		EndedAt:       r.EndedAt,
		PricePerMin:   r.PricePerMin,
		WaitingPerMin: r.WaitingPerMin,
		TotalCost:     r.TotalCost,
		Status:        r.Status,
	}
}
