package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"carsharing/internal/models"
)

func TestRequireRole(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := RequireRole(models.RoleManager, models.RoleAdmin)(ok)

	cases := []struct {
		name string
		role string
		want int
	}{
		{"клиент", models.RoleClient, http.StatusForbidden},
		{"менеджер", models.RoleManager, http.StatusOK},
		{"админ", models.RoleAdmin, http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req = req.WithContext(WithUser(req.Context(), models.User{ID: 1, Role: tc.role}))
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Errorf("код %d, ожидался %d", rec.Code, tc.want)
			}
		})
	}

	t.Run("без пользователя в контексте", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("код %d, ожидался 401", rec.Code)
		}
	})
}
