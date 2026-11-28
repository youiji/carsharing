package e2e

import (
	"net/http"
	"strings"
	"testing"
)

func TestWeb(t *testing.T) {
	e := newEnv(t)

	get := func(t *testing.T, path string) *http.Response {
		t.Helper()
		resp, err := e.srv.Client().Get(e.srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp
	}

	t.Run("главная страница", func(t *testing.T) {
		resp := get(t, "/")
		if resp.StatusCode != http.StatusOK || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html") {
			t.Errorf("статус %d, тип %q", resp.StatusCode, resp.Header.Get("Content-Type"))
		}
		if !strings.Contains(resp.Header.Get("Content-Security-Policy"), "script-src 'self'") {
			t.Error("нет заголовка CSP")
		}
	})

	t.Run("неизвестная страница — 404 HTML", func(t *testing.T) {
		resp := get(t, "/nope")
		if resp.StatusCode != http.StatusNotFound || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html") {
			t.Errorf("статус %d, тип %q", resp.StatusCode, resp.Header.Get("Content-Type"))
		}
	})

	t.Run("статика отдаётся", func(t *testing.T) {
		if resp := get(t, "/static/js/api.js"); resp.StatusCode != http.StatusOK {
			t.Errorf("статус %d", resp.StatusCode)
		}
	})

	t.Run("список файлов каталога не отдаётся", func(t *testing.T) {
		if resp := get(t, "/static/js/"); resp.StatusCode != http.StatusNotFound {
			t.Errorf("статус %d, ожидался 404", resp.StatusCode)
		}
	})

	t.Run("неизвестный путь API — 404 JSON", func(t *testing.T) {
		r := e.request(t, http.MethodGet, "/api/nope", nil)
		if r.status != http.StatusNotFound || errorCode(r) != "not_found" {
			t.Errorf("получили %d %q", r.status, errorCode(r))
		}
	})

	t.Run("неверный метод API — 405 JSON", func(t *testing.T) {
		r := e.request(t, http.MethodDelete, "/api/health", nil)
		if r.status != http.StatusMethodNotAllowed || errorCode(r) != "method_not_allowed" {
			t.Errorf("получили %d %q", r.status, errorCode(r))
		}
	})
}
