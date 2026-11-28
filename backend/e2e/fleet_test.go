package e2e

import (
	"fmt"
	"net/http"
	"testing"
)

func TestFleet(t *testing.T) {
	e := newEnv(t)
	manager := e.login(t, "manager1@carsharing.local", "Manager123!")
	client := e.login(t, "dmitry.ivanov@example.com", "Client123!")

	newCar := map[string]any{
		"tariff_id": 1, "brand_model": "Lada Granta", "plate_number": "a777aa799",
		"fuel_level": 50, "latitude": 55.75, "longitude": 37.61,
	}

	var carID any
	t.Run("менеджер добавляет машину, номер нормализован", func(t *testing.T) {
		r := e.requestAs(t, manager, http.MethodPost, "/api/cars", newCar)
		if r.status != http.StatusCreated || r.body["plate_number"] != "А777АА799" || r.body["status"] != "available" {
			t.Fatalf("получили %d %v", r.status, r.body)
		}
		carID = r.body["id"]
	})

	cases := []struct {
		name       string
		token      string
		method     string
		path       string
		body       map[string]any
		wantStatus int
		wantCode   string
	}{
		{"клиент не добавляет", client, http.MethodPost, "/api/cars", newCar, http.StatusForbidden, "forbidden"},
		{"дубликат номера", manager, http.MethodPost, "/api/cars", newCar, http.StatusConflict, "plate_taken"},
		{"несуществующий тариф", manager, http.MethodPost, "/api/cars",
			map[string]any{"tariff_id": 999, "brand_model": "X", "plate_number": "Х001ХХ77", "fuel_level": 1, "latitude": 1, "longitude": 1},
			http.StatusBadRequest, "validation_error"},
		{"топливо 150", manager, http.MethodPost, "/api/cars",
			map[string]any{"tariff_id": 1, "brand_model": "X", "plate_number": "Х002ХХ77", "fuel_level": 150, "latitude": 1, "longitude": 1},
			http.StatusBadRequest, "validation_error"},
		{"статус booked вручную", manager, http.MethodPatch, "/api/cars/1/status", map[string]any{"status": "booked"}, http.StatusBadRequest, "validation_error"},
		{"забронированную машину не трогать", manager, http.MethodPatch, "/api/cars/4/status", map[string]any{"status": "maintenance"}, http.StatusConflict, "invalid_status_transition"},
		{"списанную сразу на обслуживание нельзя", manager, http.MethodPatch, "/api/cars/8/status", map[string]any{"status": "maintenance"}, http.StatusConflict, "invalid_status_transition"},
		{"машину с историей не удалить", manager, http.MethodDelete, "/api/cars/1", nil, http.StatusConflict, "has_related_records"},
		{"латинская L в номере", manager, http.MethodPost, "/api/cars",
			map[string]any{"tariff_id": 1, "brand_model": "X", "plate_number": "L123AA77", "fuel_level": 1, "latitude": 1, "longitude": 1},
			http.StatusBadRequest, "validation_error"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := e.requestAs(t, tc.token, tc.method, tc.path, tc.body)
			if r.status != tc.wantStatus || errorCode(r) != tc.wantCode {
				t.Errorf("получили %d %q, ожидали %d %q, тело %v", r.status, errorCode(r), tc.wantStatus, tc.wantCode, r.body)
			}
		})
	}

	t.Run("на обслуживание и обратно", func(t *testing.T) {
		for _, status := range []string{"maintenance", "available"} {
			r := e.requestAs(t, manager, http.MethodPatch, fmt.Sprintf("/api/cars/%v/status", carID), map[string]any{"status": status})
			if r.status != http.StatusOK || r.body["status"] != status {
				t.Fatalf("переход в %s: %d %v", status, r.status, r.body)
			}
		}
	})

	t.Run("новую машину без истории можно удалить", func(t *testing.T) {
		r := e.requestAs(t, manager, http.MethodDelete, fmt.Sprintf("/api/cars/%v", carID), nil)
		if r.status != http.StatusNoContent {
			t.Errorf("статус %d, ожидался 204", r.status)
		}
	})
}

func TestTariffsAdmin(t *testing.T) {
	e := newEnv(t)
	manager := e.login(t, "manager1@carsharing.local", "Manager123!")

	var id any
	t.Run("создание", func(t *testing.T) {
		r := e.requestAs(t, manager, http.MethodPost, "/api/tariffs",
			map[string]any{"name": "Премиум", "price_per_min": 25.5, "waiting_per_min": 6})
		if r.status != http.StatusCreated || r.body["price_per_min"] != 25.5 {
			t.Fatalf("получили %d %v", r.status, r.body)
		}
		id = r.body["id"]
	})

	cases := []struct {
		name       string
		method     string
		path       string
		body       map[string]any
		wantStatus int
		wantCode   string
	}{
		{"название занято в другом регистре", http.MethodPost, "/api/tariffs",
			map[string]any{"name": "премиум", "price_per_min": 1, "waiting_per_min": 1}, http.StatusConflict, "name_taken"},
		{"три знака после запятой", http.MethodPost, "/api/tariffs",
			map[string]any{"name": "Ошибка", "price_per_min": 1.234, "waiting_per_min": 1}, http.StatusBadRequest, "validation_error"},
		{"цена строкой", http.MethodPost, "/api/tariffs",
			map[string]any{"name": "Ошибка", "price_per_min": "10", "waiting_per_min": 1}, http.StatusBadRequest, "validation_error"},
		{"используемый тариф не удалить", http.MethodDelete, "/api/tariffs/1", nil, http.StatusConflict, "has_related_records"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := e.requestAs(t, manager, tc.method, tc.path, tc.body)
			if r.status != tc.wantStatus || errorCode(r) != tc.wantCode {
				t.Errorf("получили %d %q, ожидали %d %q", r.status, errorCode(r), tc.wantStatus, tc.wantCode)
			}
		})
	}

	t.Run("неиспользуемый тариф удаляется", func(t *testing.T) {
		r := e.requestAs(t, manager, http.MethodDelete, fmt.Sprintf("/api/tariffs/%v", id), nil)
		if r.status != http.StatusNoContent {
			t.Errorf("статус %d, ожидался 204", r.status)
		}
	})
}
