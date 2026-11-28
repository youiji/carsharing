package e2e

import (
	"fmt"
	"net/http"
	"testing"
)

func TestMaintenance(t *testing.T) {
	e := newEnv(t)
	m1 := e.login(t, "manager1@carsharing.local", "Manager123!")
	admin := e.login(t, "admin@carsharing.local", "Admin123!")
	client := e.login(t, "ivan.petrov@example.com", "Client123!")

	t.Run("фильтр по машине", func(t *testing.T) {
		r := e.requestAs(t, m1, http.MethodGet, "/api/maintenance?car_id=1", nil)
		if r.status != http.StatusOK || r.body["total"] != float64(1) {
			t.Fatalf("получили %d %v", r.status, r.body)
		}
	})

	t.Run("сортировка по стоимости", func(t *testing.T) {
		r := e.requestAs(t, m1, http.MethodGet, "/api/maintenance?sort=cost&order=desc", nil)
		items, _ := r.body["items"].([]any)
		if len(items) == 0 || items[0].(map[string]any)["cost"] != float64(12000) {
			t.Fatalf("первая запись не самая дорогая: %v", r.body)
		}
	})

	t.Run("поиск по описанию", func(t *testing.T) {
		r := e.requestAs(t, m1, http.MethodGet, "/api/maintenance?q=колодок", nil)
		if r.body["total"] != float64(1) {
			t.Errorf("ожидали 1 запись, тело %v", r.body)
		}
	})

	var ownID any
	t.Run("менеджер добавляет, автор из токена", func(t *testing.T) {
		r := e.requestAs(t, m1, http.MethodPost, "/api/maintenance",
			map[string]any{"car_id": 7, "description": "Замена щёток", "cost": 1500.5})
		manager, _ := r.body["manager"].(map[string]any)
		if r.status != http.StatusCreated || manager["id"] != float64(2) || r.body["cost"] != 1500.5 {
			t.Fatalf("получили %d %v", r.status, r.body)
		}
		ownID = r.body["id"]
	})

	t.Run("PATCH меняет только присланное", func(t *testing.T) {
		r := e.requestAs(t, m1, http.MethodPatch, fmt.Sprintf("/api/maintenance/%v", ownID),
			map[string]any{"description": "Замена щёток стеклоочистителя"})
		if r.status != http.StatusOK || r.body["cost"] != 1500.5 || r.body["description"] != "Замена щёток стеклоочистителя" {
			t.Fatalf("получили %d %v", r.status, r.body)
		}
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
		{"клиенту журнал недоступен", client, http.MethodGet, "/api/maintenance", nil, http.StatusForbidden, "forbidden"},
		{"несуществующая машина", m1, http.MethodPost, "/api/maintenance",
			map[string]any{"car_id": 999999, "description": "x", "cost": 1}, http.StatusBadRequest, "validation_error"},
		{"пустое описание", m1, http.MethodPost, "/api/maintenance",
			map[string]any{"car_id": 1, "description": "   ", "cost": 1}, http.StatusBadRequest, "validation_error"},
		{"без стоимости", m1, http.MethodPost, "/api/maintenance",
			map[string]any{"car_id": 1, "description": "x"}, http.StatusBadRequest, "validation_error"},
		{"отрицательная стоимость", m1, http.MethodPost, "/api/maintenance",
			map[string]any{"car_id": 1, "description": "x", "cost": -1}, http.StatusBadRequest, "validation_error"},
		{"пустой PATCH", m1, http.MethodPatch, "/api/maintenance/1", map[string]any{}, http.StatusBadRequest, "validation_error"},
		{"чужую запись не править", m1, http.MethodPatch, "/api/maintenance/3", map[string]any{"cost": 1}, http.StatusForbidden, "forbidden"},
		{"чужую запись не удалить", m1, http.MethodDelete, "/api/maintenance/3", nil, http.StatusForbidden, "forbidden"},
		{"несуществующая запись", m1, http.MethodPatch, "/api/maintenance/999999", map[string]any{"cost": 1}, http.StatusNotFound, "not_found"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := e.requestAs(t, tc.token, tc.method, tc.path, tc.body)
			if r.status != tc.wantStatus || errorCode(r) != tc.wantCode {
				t.Errorf("получили %d %q, ожидали %d %q, тело %v", r.status, errorCode(r), tc.wantStatus, tc.wantCode, r.body)
			}
		})
	}

	t.Run("manager_id из тела игнорируется", func(t *testing.T) {
		r := e.requestAs(t, m1, http.MethodPost, "/api/maintenance",
			map[string]any{"car_id": 1, "description": "Попытка подмены", "cost": 1, "manager_id": 3})
		manager, _ := r.body["manager"].(map[string]any)
		if r.status != http.StatusCreated || manager["id"] != float64(2) {
			t.Fatalf("автор подменён или ошибка: %d %v", r.status, r.body)
		}
	})

	t.Run("админ правит и удаляет чужую", func(t *testing.T) {
		if r := e.requestAs(t, admin, http.MethodPatch, "/api/maintenance/3", map[string]any{"description": "Проверено"}); r.status != http.StatusOK {
			t.Fatalf("PATCH: %d %v", r.status, r.body)
		}
		if r := e.requestAs(t, admin, http.MethodDelete, "/api/maintenance/3", nil); r.status != http.StatusNoContent {
			t.Fatalf("DELETE: %d %v", r.status, r.body)
		}
	})

	t.Run("менеджер удаляет свою, повторно 404", func(t *testing.T) {
		path := fmt.Sprintf("/api/maintenance/%v", ownID)
		if r := e.requestAs(t, m1, http.MethodDelete, path, nil); r.status != http.StatusNoContent {
			t.Fatalf("первое удаление: %d %v", r.status, r.body)
		}
		if r := e.requestAs(t, m1, http.MethodDelete, path, nil); r.status != http.StatusNotFound {
			t.Errorf("повторное удаление: %d, ожидали 404", r.status)
		}
	})
}
