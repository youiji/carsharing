package e2e

import (
	"net/http"
	"testing"
	"time"

	"carsharing/internal/util"
)

func TestMe(t *testing.T) {
	e := newEnv(t)
	tokens := util.NewTokenManager(testSecret, 30*time.Minute)

	t.Run("свой профиль", func(t *testing.T) {
		token := e.login(t, "ivan.petrov@example.com", "Client123!")
		r := e.requestAs(t, token, http.MethodGet, "/api/me", nil)
		if r.status != http.StatusOK || r.body["email"] != "ivan.petrov@example.com" {
			t.Errorf("получили %d %v", r.status, r.body)
		}
	})

	t.Run("роль берётся из БД, а не из токена", func(t *testing.T) {
		forged, _ := tokens.Issue(4, "admin") // Иван, но в токене роль admin
		r := e.requestAs(t, forged, http.MethodGet, "/api/me", nil)
		if r.body["role"] != "client" {
			t.Errorf("роль %v, ожидалась client из БД", r.body["role"])
		}
	})

	expired, _ := util.NewTokenManager(testSecret, -time.Minute).Issue(4, "client")
	ghost, _ := tokens.Issue(9999, "admin")

	cases := []struct {
		name  string
		token string
	}{
		{"без токена", ""},
		{"мусор вместо токена", "not-a-jwt"},
		{"просроченный токен", expired},
		{"пользователя нет в БД", ghost},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := e.requestAs(t, tc.token, http.MethodGet, "/api/me", nil)
			if r.status != http.StatusUnauthorized || errorCode(r) != "unauthorized" {
				t.Errorf("получили %d %q, ожидали 401 unauthorized", r.status, errorCode(r))
			}
		})
	}
}

func TestChangePassword(t *testing.T) {
	e := newEnv(t)
	const email = "anna.sidorova@example.com"
	token := e.login(t, email, "Client123!")

	cases := []struct {
		name       string
		body       map[string]any
		wantStatus int
		wantCode   string
	}{
		{"неверный текущий пароль", map[string]any{"current_password": "Wrong123!", "new_password": "NewPass456!"}, http.StatusBadRequest, "wrong_password"},
		{"слабый новый пароль", map[string]any{"current_password": "Client123!", "new_password": "123"}, http.StatusBadRequest, "validation_error"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := e.requestAs(t, token, http.MethodPut, "/api/me/password", tc.body)
			if r.status != tc.wantStatus || errorCode(r) != tc.wantCode {
				t.Errorf("получили %d %q, ожидали %d %q", r.status, errorCode(r), tc.wantStatus, tc.wantCode)
			}
		})
	}

	t.Run("без токена", func(t *testing.T) {
		r := e.request(t, http.MethodPut, "/api/me/password", map[string]any{"current_password": "Client123!", "new_password": "NewPass456!"})
		if r.status != http.StatusUnauthorized {
			t.Errorf("статус %d, ожидался 401", r.status)
		}
	})

	t.Run("успешная смена", func(t *testing.T) {
		r := e.requestAs(t, token, http.MethodPut, "/api/me/password", map[string]any{"current_password": "Client123!", "new_password": "NewPass456!"})
		if r.status != http.StatusNoContent {
			t.Fatalf("статус %d, ожидался 204, тело %v", r.status, r.body)
		}
		old := e.request(t, http.MethodPost, "/api/auth/login", map[string]any{"email": email, "password": "Client123!"})
		if old.status != http.StatusUnauthorized {
			t.Errorf("старый пароль: статус %d, ожидался 401", old.status)
		}
		e.login(t, email, "NewPass456!") // упадёт сам, если новый пароль не подошёл
	})
}
