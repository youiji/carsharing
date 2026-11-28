package e2e

import (
	"net/http"
	"testing"
)

func TestUsersAdmin(t *testing.T) {
	e := newEnv(t)
	admin := e.login(t, "admin@carsharing.local", "Admin123!")
	m1 := e.login(t, "manager1@carsharing.local", "Manager123!")
	m2 := e.login(t, "manager2@carsharing.local", "Manager123!")

	lists := []struct {
		name  string
		query string
		want  float64
	}{
		{"все", "", 9},
		{"только менеджеры", "?role=manager", 2},
		{"поиск без учёта регистра", "?q=VOLKOVA", 1},
	}
	for _, tc := range lists {
		t.Run("список: "+tc.name, func(t *testing.T) {
			r := e.requestAs(t, admin, http.MethodGet, "/api/users"+tc.query, nil)
			if r.status != http.StatusOK || r.body["total"] != tc.want {
				t.Errorf("получили %d total=%v, ожидали %v", r.status, r.body["total"], tc.want)
			}
		})
	}

	cases := []struct {
		name       string
		token      string
		method     string
		path       string
		body       map[string]any
		wantStatus int
		wantCode   string
	}{
		{"менеджеру список недоступен", m1, http.MethodGet, "/api/users", nil, http.StatusForbidden, "forbidden"},
		{"неизвестная роль в фильтре", admin, http.MethodGet, "/api/users?role=root", nil, http.StatusBadRequest, "validation_error"},
		{"неизвестная роль", admin, http.MethodPatch, "/api/users/6/role", map[string]any{"role": "root"}, http.StatusBadRequest, "validation_error"},
		{"свою роль нельзя", admin, http.MethodPatch, "/api/users/1/role", map[string]any{"role": "client"}, http.StatusForbidden, "cannot_change_own_role"},
		{"роль при открытой поездке нельзя", admin, http.MethodPatch, "/api/users/4/role", map[string]any{"role": "manager"}, http.StatusConflict, "user_has_active_rental"},
		{"та же роль при поездке можно", admin, http.MethodPatch, "/api/users/4/role", map[string]any{"role": "client"}, http.StatusOK, ""},
		{"несуществующий пользователь", admin, http.MethodPatch, "/api/users/999999/role", map[string]any{"role": "client"}, http.StatusNotFound, "not_found"},
		{"данные при открытой поездке нельзя", admin, http.MethodPatch, "/api/users/4", map[string]any{"full_name": "Петров И. С."}, http.StatusConflict, "user_has_active_rental"},
		{"пустой PATCH", admin, http.MethodPatch, "/api/users/6", map[string]any{}, http.StatusBadRequest, "validation_error"},
		{"чужое ВУ", admin, http.MethodPatch, "/api/users/6", map[string]any{"driver_license": "7712345678"}, http.StatusConflict, "license_taken"},
		{"чужой email", admin, http.MethodPatch, "/api/users/6", map[string]any{"email": "ANNA.SIDOROVA@example.com"}, http.StatusConflict, "email_taken"},
		{"себя удалить нельзя", admin, http.MethodDelete, "/api/users/1", nil, http.StatusForbidden, "cannot_delete_self"},
		{"клиента с историей не удалить", admin, http.MethodDelete, "/api/users/4", nil, http.StatusConflict, "has_related_records"},
		{"менеджера с журналом не удалить", admin, http.MethodDelete, "/api/users/2", nil, http.StatusConflict, "has_related_records"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := e.requestAs(t, tc.token, tc.method, tc.path, tc.body)
			if r.status != tc.wantStatus || errorCode(r) != tc.wantCode {
				t.Errorf("получили %d %q, ожидали %d %q, тело %v", r.status, errorCode(r), tc.wantStatus, tc.wantCode, r.body)
			}
		})
	}

	t.Run("смена ФИО и ВУ клиенту (как обращение в поддержку)", func(t *testing.T) {
		r := e.requestAs(t, admin, http.MethodPatch, "/api/users/6",
			map[string]any{"full_name": "Иванов-Петров Дмитрий Олегович", "driver_license": "7799000111"})
		if r.status != http.StatusOK || r.body["driver_license"] != "7799 000111" {
			t.Fatalf("получили %d %v", r.status, r.body)
		}
	})

	t.Run("разжалованный менеджер сразу теряет доступ", func(t *testing.T) {
		if r := e.requestAs(t, m2, http.MethodGet, "/api/maintenance", nil); r.status != http.StatusOK {
			t.Fatalf("до разжалования: %d", r.status)
		}
		if r := e.requestAs(t, admin, http.MethodPatch, "/api/users/3/role", map[string]any{"role": "client"}); r.status != http.StatusOK {
			t.Fatalf("смена роли: %d %v", r.status, r.body)
		}
		if r := e.requestAs(t, m2, http.MethodGet, "/api/maintenance", nil); r.status != http.StatusForbidden {
			t.Errorf("старый токен всё ещё пускает: %d", r.status)
		}
	})

	t.Run("пользователя без истории можно удалить, повторно 404", func(t *testing.T) {
		if r := e.requestAs(t, admin, http.MethodDelete, "/api/users/9", nil); r.status != http.StatusNoContent {
			t.Fatalf("удаление: %d %v", r.status, r.body)
		}
		if r := e.requestAs(t, admin, http.MethodDelete, "/api/users/9", nil); r.status != http.StatusNotFound {
			t.Errorf("повторно: %d", r.status)
		}
	})
}
