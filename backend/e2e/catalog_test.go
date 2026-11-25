package e2e

import (
	"net/http"
	"testing"
)

func TestCatalog(t *testing.T) {
	e := newEnv(t)
	client := e.login(t, "dmitry.ivanov@example.com", "Client123!")
	manager := e.login(t, "manager1@carsharing.local", "Manager123!")

	total := func(r response) float64 {
		n, _ := r.body["total"].(float64)
		return n
	}
	items := func(r response) []any {
		list, _ := r.body["items"].([]any)
		return list
	}

	t.Run("клиент видит только свободные", func(t *testing.T) {
		r := e.requestAs(t, client, http.MethodGet, "/api/cars", nil)
		if r.status != http.StatusOK || total(r) != 12 {
			t.Fatalf("получили %d, total %v, ожидали 200 и 12", r.status, total(r))
		}
		for _, it := range items(r) {
			car, _ := it.(map[string]any)
			if car["status"] != "available" {
				t.Errorf("клиенту показана машина в статусе %v", car["status"])
			}
		}
	})

	t.Run("клиент не обходит фильтр через status", func(t *testing.T) {
		r := e.requestAs(t, client, http.MethodGet, "/api/cars?status=in_use", nil)
		if total(r) != 12 {
			t.Errorf("total %v, ожидали 12 свободных", total(r))
		}
	})

	cases := []struct {
		name  string
		token string
		query string
		want  float64
	}{
		{"менеджер видит все", manager, "", 18},
		{"менеджер фильтрует по статусу", manager, "?status=maintenance", 1},
		{"поиск по марке без учёта регистра", manager, "?q=kia", 3},
		{"поиск по номеру латиницей", manager, "?q=a101", 1},
		{"клиент ищет kia", client, "?q=kia", 1},
		{"фильтр по тарифу", manager, "?tariff_id=3", 4},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := e.requestAs(t, tc.token, http.MethodGet, "/api/cars"+tc.query, nil)
			if r.status != http.StatusOK || total(r) != tc.want {
				t.Errorf("получили %d, total %v, ожидали 200 и %v", r.status, total(r), tc.want)
			}
		})
	}

	t.Run("сортировка по цене по убыванию", func(t *testing.T) {
		r := e.requestAs(t, manager, http.MethodGet, "/api/cars?sort=price_per_min&order=desc", nil)
		list := items(r)
		if len(list) == 0 {
			t.Fatalf("пустой список, тело %v", r.body)
		}
		first, _ := list[0].(map[string]any)
		tariff, _ := first["tariff"].(map[string]any)
		if tariff["price_per_min"] != 19.9 {
			t.Errorf("первая цена %v, ожидалась 19.9", tariff["price_per_min"])
		}
	})

	bad := []struct {
		name  string
		query string
	}{
		{"инъекция через sort", "?sort=id3BDROP%20TABLE%20cars"},
		{"неверный order", "?order=sideways"},
		{"неизвестный статус", "?status=flying"},
		{"tariff_id не число", "?tariff_id=abc"},
	}
	for _, tc := range bad {
		t.Run(tc.name, func(t *testing.T) {
			r := e.requestAs(t, manager, http.MethodGet, "/api/cars"+tc.query, nil)
			if r.status != http.StatusBadRequest || errorCode(r) != "validation_error" {
				t.Errorf("получили %d %q, ожидали 400 validation_error", r.status, errorCode(r))
			}
		})
	}

	t.Run("без токена", func(t *testing.T) {
		r := e.request(t, http.MethodGet, "/api/cars", nil)
		if r.status != http.StatusUnauthorized {
			t.Errorf("статус %d, ожидался 401", r.status)
		}
	})
}
