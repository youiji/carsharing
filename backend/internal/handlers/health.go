package handlers

import (
	"context"
	"log"
	"net/http"
	"time"

	"carsharing/internal/util"

	"github.com/jackc/pgx/v5/pgxpool"
)

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
