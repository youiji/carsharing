package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"carsharing/internal/config"
	"carsharing/internal/repository"
	"carsharing/internal/util"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	pool, err := repository.NewPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	users := repository.NewUserRepository(pool)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", healthHandler(pool))
	mux.HandleFunc("GET /api/ping", pingHandler)
	mux.HandleFunc("GET /api/debug/user", debugUserHandler(users))

	log.Println("сервер запущен на", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, mux); err != nil {
		log.Fatal(err)
	}
}

func healthHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			log.Printf("health: %v", err)
			util.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status":   "degraded",
				"database": "unavailable",
			})
			return
		}
		util.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "database": "ok"})
	}
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	util.WriteJSON(w, http.StatusOK, map[string]string{"message": "pong"})
}

func debugUserHandler(users *repository.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email := r.URL.Query().Get("email")
		u, err := users.GetByEmail(r.Context(), email)
		if errors.Is(err, repository.ErrNotFound) {
			util.WriteError(w, r, util.ErrNotFound)
			return
		}
		if err != nil {
			util.WriteError(w, r, err)
			return
		}
		util.WriteJSON(w, http.StatusOK, map[string]any{
			"id":             u.ID,
			"email":          u.Email,
			"full_name":      u.FullName,
			"driver_license": u.DriverLicense,
			"role":           u.Role,
			"created_at":     u.CreatedAt,
		})
	}
}
