package services

import (
	"testing"
	"time"

	"carsharing/internal/util"
)

func TestCalcRentalCost(t *testing.T) {
	base := time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC)
	const price, waiting = util.Money(890), util.Money(200) // 8.90 и 2.00

	cases := []struct {
		name               string
		booked, start, end time.Time
		want               util.Money
	}{
		{"ровно 10 минут поездки, без ожидания", base, base, base.Add(10 * time.Minute), 10 * price},
		{"10 минут и 1 секунда — округление вверх", base, base, base.Add(10*time.Minute + time.Second), 11 * price},
		{"мгновенная поездка — минимум 1 минута", base, base, base, price},
		{"ожидание 3 минуты и поездка 5", base, base.Add(3 * time.Minute), base.Add(8 * time.Minute), 3*waiting + 5*price},
		{"ожидание 30 секунд — 1 минута", base, base.Add(30 * time.Second), base.Add(30*time.Second + 2*time.Minute), waiting + 2*price},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CalcRentalCost(tc.booked, tc.start, tc.end, price, waiting)
			if got != tc.want {
				t.Errorf("получили %s, ожидали %s", got, tc.want)
			}
		})
	}
}

func TestCancellationFee(t *testing.T) {
	base := time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC)
	const waiting = util.Money(200)

	cases := []struct {
		name     string
		cancelAt time.Time
		want     util.Money
	}{
		{"сразу после бронирования", base, 0},
		{"через 5 минут", base.Add(5 * time.Minute), 5 * waiting},
		{"через 4 минуты 1 секунду", base.Add(4*time.Minute + time.Second), 5 * waiting},
		{"позже срока — не больше 20 минут", base.Add(45 * time.Minute), 20 * waiting},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CancellationFee(base, tc.cancelAt, waiting); got != tc.want {
				t.Errorf("получили %s, ожидали %s", got, tc.want)
			}
		})
	}
}
