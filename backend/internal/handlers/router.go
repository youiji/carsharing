package handlers

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"carsharing/internal/services"
)

type Deps struct {
	DB   *pgxpool.Pool
	Auth *services.AuthService
}

func NewRouter(d Deps) http.Handler {
	mux := http.NewServeMux()
	auth := &AuthHandler{svc: d.Auth}

	mux.HandleFunc("GET /api/health", healthHandler(d.DB))
	mux.HandleFunc("POST /api/auth/register", auth.Register)
	mux.HandleFunc("POST /api/auth/login", auth.Login)
	return mux
}
