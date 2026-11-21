package handlers

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"carsharing/internal/middleware"
	"carsharing/internal/services"
	"carsharing/internal/util"
)

type Deps struct {
	DB     *pgxpool.Pool
	Tokens *util.TokenManager
	Users  middleware.UserLoader
	Auth   *services.AuthService
}

func NewRouter(d Deps) http.Handler {
	mux := http.NewServeMux()
	auth := &AuthHandler{svc: d.Auth}
	authed := middleware.Auth(d.Tokens, d.Users)

	mux.HandleFunc("GET /api/health", healthHandler(d.DB))
	mux.HandleFunc("POST /api/auth/register", auth.Register)
	mux.HandleFunc("POST /api/auth/login", auth.Login)
	mux.Handle("GET /api/me", authed(http.HandlerFunc(auth.Me)))
	mux.Handle("PUT /api/me/password", authed(http.HandlerFunc(auth.ChangePassword)))

	return mux
}
