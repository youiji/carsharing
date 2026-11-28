package dto

import (
	"carsharing/internal/models"
	"carsharing/internal/util"
)

type List[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
}

func NewList[T any](items []T) List[T] {
	if items == nil {
		items = []T{}
	}
	return List[T]{Items: items, Total: len(items)}
}

type Tariff struct {
	ID            int64      `json:"id"`
	Name          string     `json:"name"`
	PricePerMin   util.Money `json:"price_per_min"`
	WaitingPerMin util.Money `json:"waiting_per_min"`
}

type Car struct {
	ID          int64   `json:"id"`
	BrandModel  string  `json:"brand_model"`
	PlateNumber string  `json:"plate_number"`
	FuelLevel   int     `json:"fuel_level"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Status      string  `json:"status"`
	Tariff      Tariff  `json:"tariff"`
}

func NewTariff(t models.Tariff) Tariff {
	return Tariff{ID: t.ID, Name: t.Name, PricePerMin: t.PricePerMin, WaitingPerMin: t.WaitingPerMin}
}

func NewCar(c models.Car) Car {
	return Car{
		ID:          c.ID,
		BrandModel:  c.BrandModel,
		PlateNumber: c.PlateNumber,
		FuelLevel:   c.FuelLevel,
		Latitude:    c.Latitude,
		Longitude:   c.Longitude,
		Status:      c.Status,
		Tariff:      NewTariff(c.Tariff),
	}
}

func NewCars(cs []models.Car) []Car {
	out := make([]Car, len(cs))
	for i, c := range cs {
		out[i] = NewCar(c)
	}
	return out
}

type CarRequest struct {
	TariffID    *int64   `json:"tariff_id"`
	BrandModel  string   `json:"brand_model"`
	PlateNumber string   `json:"plate_number"`
	FuelLevel   *int     `json:"fuel_level"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
}

type CarStatusRequest struct {
	Status string `json:"status"`
}

type TariffRequest struct {
	Name          string      `json:"name"`
	PricePerMin   *util.Money `json:"price_per_min"`
	WaitingPerMin *util.Money `json:"waiting_per_min"`
}
