package e2e

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"
)

func TestBook(t *testing.T) {
	e := newEnv(t)
	dmitry := e.login(t, "dmitry.ivanov@example.com", "Client123!")
	noLicense := e.login(t, "new.client@example.com", "Client123!")
	manager := e.login(t, "manager1@carsharing.local", "Manager123!")

	cases := []struct {
		name       string
		token      string
		body       map[string]any
		wantStatus int
		wantCode   string
	}{
		{"нет car_id", dmitry, map[string]any{}, http.StatusBadRequest, "validation_error"},
		{"машины нет", dmitry, map[string]any{"car_id": 9999}, http.StatusNotFound, "not_found"},
		{"машина на обслуживании", dmitry, map[string]any{"car_id": 7}, http.StatusConflict, "car_unavailable"},
		{"менеджер не бронирует", manager, map[string]any{"car_id": 2}, http.StatusForbidden, "forbidden"},
		{"клиент без ВУ", noLicense, map[string]any{"car_id": 2}, http.StatusForbidden, "license_required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := e.requestAs(t, tc.token, http.MethodPost, "/api/rentals", tc.body)
			if r.status != tc.wantStatus || errorCode(r) != tc.wantCode {
				t.Errorf("получили %d %q, ожидали %d %q", r.status, errorCode(r), tc.wantStatus, tc.wantCode)
			}
		})
	}

	// Второй свободный клиент для гонки
	reg := e.request(t, http.MethodPost, "/api/auth/register", map[string]any{
		"email": "racer@example.com", "password": "Racer123!", "full_name": "Гонщик Тест", "driver_license": "5555000111",
	})
	if reg.status != http.StatusCreated {
		t.Fatalf("регистрация: %d %v", reg.status, reg.body)
	}
	racer := e.login(t, "racer@example.com", "Racer123!")

	t.Run("два клиента одновременно бронируют одну машину", func(t *testing.T) {
		tokens := []string{dmitry, racer}
		results := make([]response, len(tokens))

		var wg sync.WaitGroup
		for i, token := range tokens {
			wg.Add(1)
			go func() {
				defer wg.Done()
				results[i] = e.requestAs(t, token, http.MethodPost, "/api/rentals", map[string]any{"car_id": 2})
			}()
		}
		wg.Wait()

		created, conflicts := 0, 0
		for _, r := range results {
			switch {
			case r.status == http.StatusCreated:
				created++
				if r.body["status"] != "booked" || r.body["price_per_min"] != 8.9 || r.body["expires_at"] == nil {
					t.Errorf("неверная аренда: %v", r.body)
				}
			case r.status == http.StatusConflict && errorCode(r) == "car_unavailable":
				conflicts++
			default:
				t.Errorf("неожиданный ответ %d %v", r.status, r.body)
			}
		}
		if created != 1 || conflicts != 1 {
			t.Fatalf("создано %d, конфликтов %d, ожидали 1 и 1", created, conflicts)
		}
	})

	t.Run("машина стала booked", func(t *testing.T) {
		r := e.requestAs(t, manager, http.MethodGet, "/api/cars/2", nil)
		if r.body["status"] != "booked" {
			t.Errorf("статус машины %v, ожидался booked", r.body["status"])
		}
	})

	t.Run("второе бронирование у одного клиента", func(t *testing.T) {
		statuses := []int{}
		for _, token := range []string{dmitry, racer} {
			r := e.requestAs(t, token, http.MethodPost, "/api/rentals", map[string]any{"car_id": 5})
			statuses = append(statuses, r.status)
			if r.status == http.StatusCreated {
				return // это был проигравший гонку: он свободен и успешно забронировал машину 5
			}
			if errorCode(r) != "user_has_active_rental" {
				t.Errorf("код %q, ожидался user_has_active_rental", errorCode(r))
			}
		}
		t.Errorf("ни один не смог забронировать машину 5: %v", statuses)
	})
}

func TestRentalFlow(t *testing.T) {
	e := newEnv(t)
	dmitry := e.login(t, "dmitry.ivanov@example.com", "Client123!")
	ivan := e.login(t, "ivan.petrov@example.com", "Client123!")
	maria := e.login(t, "maria.volkova@example.com", "Client123!")
	manager := e.login(t, "manager1@carsharing.local", "Manager123!")

	path := func(id any, action string) string { return fmt.Sprintf("/api/rentals/%v/%s", id, action) }

	booked := e.requestAs(t, dmitry, http.MethodPost, "/api/rentals", map[string]any{"car_id": 1})
	if booked.status != http.StatusCreated {
		t.Fatalf("бронирование: %d %v", booked.status, booked.body)
	}
	id := booked.body["id"]

	t.Run("нельзя завершить бронирование", func(t *testing.T) {
		r := e.requestAs(t, dmitry, http.MethodPost, path(id, "finish"), nil)
		if r.status != http.StatusConflict || errorCode(r) != "invalid_status_transition" {
			t.Errorf("получили %d %q", r.status, errorCode(r))
		}
	})

	t.Run("старт", func(t *testing.T) {
		r := e.requestAs(t, dmitry, http.MethodPost, path(id, "start"), nil)
		if r.status != http.StatusOK || r.body["status"] != "active" || r.body["started_at"] == nil {
			t.Fatalf("получили %d %v", r.status, r.body)
		}
	})

	t.Run("чужую поездку не завершить", func(t *testing.T) {
		r := e.requestAs(t, ivan, http.MethodPost, path(id, "finish"), nil)
		if r.status != http.StatusForbidden {
			t.Errorf("статус %d, ожидался 403", r.status)
		}
	})

	t.Run("одна координата без второй", func(t *testing.T) {
		r := e.requestAs(t, dmitry, http.MethodPost, path(id, "finish"), map[string]any{"latitude": 55.8})
		if r.status != http.StatusBadRequest {
			t.Errorf("статус %d, ожидался 400", r.status)
		}
	})

	t.Run("завершение", func(t *testing.T) {
		r := e.requestAs(t, dmitry, http.MethodPost, path(id, "finish"), map[string]any{"latitude": 55.8, "longitude": 37.5})
		if r.status != http.StatusOK || r.body["status"] != "completed" {
			t.Fatalf("получили %d %v", r.status, r.body)
		}
		// 1 минута ожидания (2.00) + минимум 1 минута поездки (8.90)
		if r.body["total_cost"] != 10.9 {
			t.Errorf("стоимость %v, ожидалась 10.9", r.body["total_cost"])
		}
		car := e.requestAs(t, manager, http.MethodGet, "/api/cars/1", nil)
		if car.body["status"] != "available" || car.body["latitude"] != 55.8 {
			t.Errorf("машина после поездки: %v", car.body)
		}
	})

	t.Run("повторное завершение", func(t *testing.T) {
		r := e.requestAs(t, dmitry, http.MethodPost, path(id, "finish"), nil)
		if r.status != http.StatusConflict {
			t.Errorf("статус %d, ожидался 409", r.status)
		}
	})

	t.Run("клиент отменяет своё бронирование — платно", func(t *testing.T) {
		r := e.requestAs(t, maria, http.MethodPost, path(13, "cancel"), nil)
		cost, _ := r.body["total_cost"].(float64)
		if r.status != http.StatusOK || r.body["status"] != "cancelled" || cost <= 0 {
			t.Errorf("получили %d %v", r.status, r.body)
		}
	})

	t.Run("менеджер отменяет чужое бронирование — бесплатно", func(t *testing.T) {
		r := e.requestAs(t, manager, http.MethodPost, path(14, "cancel"), nil)
		if r.status != http.StatusOK || r.body["total_cost"] != 0.0 {
			t.Errorf("получили %d %v", r.status, r.body)
		}
	})

	t.Run("несуществующая аренда", func(t *testing.T) {
		r := e.requestAs(t, dmitry, http.MethodPost, path(9999, "start"), nil)
		if r.status != http.StatusNotFound {
			t.Errorf("статус %d, ожидался 404", r.status)
		}
	})
}

func (e *env) exec(t *testing.T, sql string, args ...any) {
	t.Helper()
	if _, err := e.pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("exec: %v", err)
	}
}

func TestRentalLists(t *testing.T) {
	e := newEnv(t)
	ivan := e.login(t, "ivan.petrov@example.com", "Client123!")
	anna := e.login(t, "anna.sidorova@example.com", "Client123!")
	manager := e.login(t, "manager1@carsharing.local", "Manager123!")

	total := func(r response) float64 { n, _ := r.body["total"].(float64); return n }

	cases := []struct {
		name   string
		token  string
		path   string
		status int
		total  float64
	}{
		{"мои аренды Ивана", ivan, "/api/rentals/my", http.StatusOK, 3},
		{"мои завершённые", ivan, "/api/rentals/my?status=completed", http.StatusOK, 2},
		{"все аренды у менеджера", manager, "/api/rentals", http.StatusOK, 14},
		{"поиск по ФИО", manager, "/api/rentals?q=петров", http.StatusOK, 3},
		{"фильтр по машине", manager, "/api/rentals?car_id=1", http.StatusOK, 2},
		{"клиенту список всех нельзя", ivan, "/api/rentals", http.StatusForbidden, 0},
		{"менеджеру мои аренды нельзя", manager, "/api/rentals/my", http.StatusForbidden, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := e.requestAs(t, tc.token, http.MethodGet, tc.path, nil)
			if r.status != tc.status || (tc.status == http.StatusOK && total(r) != tc.total) {
				t.Errorf("получили %d, total %v, ожидали %d и %v", r.status, total(r), tc.status, tc.total)
			}
		})
	}

	t.Run("своя аренда", func(t *testing.T) {
		r := e.requestAs(t, ivan, http.MethodGet, "/api/rentals/11", nil)
		if r.status != http.StatusOK || r.body["status"] != "active" {
			t.Errorf("получили %d %v", r.status, r.body)
		}
	})
	t.Run("чужая аренда", func(t *testing.T) {
		r := e.requestAs(t, anna, http.MethodGet, "/api/rentals/11", nil)
		if r.status != http.StatusForbidden {
			t.Errorf("статус %d, ожидался 403", r.status)
		}
	})
}

func TestBookingExpiry(t *testing.T) {
	e := newEnv(t)
	maria := e.login(t, "maria.volkova@example.com", "Client123!")
	dmitry := e.login(t, "dmitry.ivanov@example.com", "Client123!")
	manager := e.login(t, "manager1@carsharing.local", "Manager123!")

	// Бронирования Марии (13) и Олега (14) «сделаны» 25 минут назад
	e.exec(t, `UPDATE rentals SET booked_at = now() - interval '25 minutes' WHERE id IN (13, 14)`)

	t.Run("старт просроченного бронирования", func(t *testing.T) {
		r := e.requestAs(t, maria, http.MethodPost, "/api/rentals/13/start", nil)
		if r.status != http.StatusConflict || errorCode(r) != "booking_expired" {
			t.Fatalf("получили %d %q", r.status, errorCode(r))
		}
		rental := e.requestAs(t, maria, http.MethodGet, "/api/rentals/13", nil)
		// 20 минут ожидания по 2.00
		if rental.body["status"] != "cancelled" || rental.body["total_cost"] != 40.0 {
			t.Errorf("аренда после просрочки: %v", rental.body)
		}
	})

	t.Run("бронирование снимает чужие просроченные брони", func(t *testing.T) {
		r := e.requestAs(t, dmitry, http.MethodPost, "/api/rentals", map[string]any{"car_id": 12})
		if r.status != http.StatusCreated {
			t.Fatalf("машина Олега не освободилась: %d %v", r.status, r.body)
		}
		oleg := e.requestAs(t, manager, http.MethodGet, "/api/rentals/14", nil)
		if oleg.body["status"] != "cancelled" {
			t.Errorf("бронирование Олега: %v", oleg.body["status"])
		}
	})
}
