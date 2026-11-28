package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"carsharing/internal/app"
	"carsharing/internal/util"
)

const testSecret = "e2e-secret-e2e-secret-e2e-secret-e2e"

type env struct {
	srv  *httptest.Server
	pool *pgxpool.Pool
}

func newEnv(t *testing.T) *env {
	t.Helper()
	url := os.Getenv("E2E_DATABASE_URL")
	if url == "" {
		t.Skip("E2E_DATABASE_URL не задана")
	}
	cfg, err := pgx.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cfg.Database, "test") {
		t.Fatalf("база %q: тесты удаляют все данные, имя базы должно содержать \"test\"", cfg.Database)
	}

	ctx := context.Background()
	resetDB(t, ctx, url)

	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := app.NewHandler(pool, util.NewTokenManager(testSecret, 30*time.Minute), "../../frontend")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(handler)

	t.Cleanup(func() {
		srv.Close()
		pool.Close()
	})
	return &env{srv: srv, pool: pool}
}

func resetDB(t *testing.T, ctx context.Context, url string) {
	t.Helper()
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)

	exec := func(name, sql string) {
		if _, err := conn.Exec(ctx, sql); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	exec("drop schema", `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`)
	exec("schema.sql", readSQL(t, "../../db/schema.sql"))
	exec("seed.sql", readSQL(t, "../../db/seed.sql"))
}

func readSQL(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), `\`) {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

type response struct {
	status int
	body   map[string]any
}

func (e *env) request(t *testing.T, method, path string, body any) response {
	t.Helper()
	return e.requestAs(t, "", method, path, body)
}

func (e *env) requestAs(t *testing.T, token, method, path string, body any) response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, e.srv.URL+path, reader)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := e.srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil && err != io.EOF {
		t.Fatalf("разбор ответа: %v", err)
	}
	return response{status: resp.StatusCode, body: out}
}

func (e *env) login(t *testing.T, email, password string) string {
	t.Helper()
	r := e.request(t, http.MethodPost, "/api/auth/login", map[string]any{"email": email, "password": password})
	if r.status != http.StatusOK {
		t.Fatalf("вход %s: статус %d, тело %v", email, r.status, r.body)
	}
	token, _ := r.body["token"].(string)
	return token
}

func errorCode(r response) string {
	errObj, _ := r.body["error"].(map[string]any)
	code, _ := errObj["code"].(string)
	return code
}

func errorMessage(r response) string {
	errObj, _ := r.body["error"].(map[string]any)
	msg, _ := errObj["message"].(string)
	return msg
}
