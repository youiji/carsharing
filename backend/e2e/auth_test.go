package e2e

import (
	"net/http"
	"strings"
	"testing"
)

func TestRegister(t *testing.T) {
	e := newEnv(t)

	t.Run("успешная регистрация", func(t *testing.T) {
		r := e.request(t, http.MethodPost, "/api/auth/register", map[string]any{
			"email":          "test1@example.com",
			"password":       "Secret123!",
			"full_name":      "Тестов Тест",
			"driver_license": "1111222333",
			"role":           "admin",
		})
		if r.status != http.StatusCreated {
			t.Fatalf("статус %d, ожидался 201, тело %v", r.status, r.body)
		}
		if r.body["role"] != "client" {
			t.Errorf("роль %v, ожидалась client", r.body["role"])
		}
		if r.body["driver_license"] != "1111 222333" {
			t.Errorf("ВУ %v, ожидалось 1111 222333", r.body["driver_license"])
		}
		if _, leaked := r.body["password_hash"]; leaked {
			t.Error("в ответе есть хеш пароля")
		}
	})

	cases := []struct {
		name       string
		body       map[string]any
		wantStatus int
		wantCode   string
	}{
		{
			name:       "email занят, другой регистр",
			body:       map[string]any{"email": "IVAN.PETROV@example.com", "password": "Secret123!", "full_name": "Двойник", "driver_license": "1111222444"},
			wantStatus: http.StatusConflict,
			wantCode:   "email_taken",
		},
		{
			name:       "ВУ занято, записано без пробела",
			body:       map[string]any{"email": "test2@example.com", "password": "Secret123!", "full_name": "Двойник", "driver_license": "7712345678"},
			wantStatus: http.StatusConflict,
			wantCode:   "license_taken",
		},
		{
			name:       "все поля неверные",
			body:       map[string]any{"email": "bad", "password": "123", "full_name": "", "driver_license": "12-34"},
			wantStatus: http.StatusBadRequest,
			wantCode:   "validation_error",
		},
		{
			name:       "пустое тело",
			body:       nil,
			wantStatus: http.StatusBadRequest,
			wantCode:   "validation_error",
		},
		{
			name:       "пароль длиннее 72 байт",
			body:       map[string]any{"email": "test3@example.com", "password": strings.Repeat("я", 40), "full_name": "Длинный пароль", "driver_license": "7271 223928"},
			wantStatus: http.StatusBadRequest,
			wantCode:   "validation_error",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := e.request(t, http.MethodPost, "/api/auth/register", tc.body)
			if r.status != tc.wantStatus || errorCode(r) != tc.wantCode {
				t.Errorf("получили %d %q, ожидали %d %q, тело %v",
					r.status, errorCode(r), tc.wantStatus, tc.wantCode, r.body)
			}
		})
	}
}
