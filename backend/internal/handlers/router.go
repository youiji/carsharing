package handlers

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"carsharing/internal/middleware"
	"carsharing/internal/models"
	"carsharing/internal/services"
	"carsharing/internal/util"
)

type Deps struct {
	DB      *pgxpool.Pool
	Tokens  *util.TokenManager
	Users   middleware.UserLoader
	Auth    *services.AuthService
	Cars    *services.CarService
	Tariffs *services.TariffService
	Rentals *services.RentalService
}

func NewRouter(d Deps) http.Handler {
	mux := http.NewServeMux()
	auth := &AuthHandler{svc: d.Auth}
	authed := middleware.Auth(d.Tokens, d.Users)
	cars := &CarHandler{svc: d.Cars}
	tariffs := &TariffHandler{svc: d.Tariffs}
	rentals := &RentalHandler{svc: d.Rentals}
	clientOnly := middleware.RequireRole(models.RoleClient)
	staffOnly := middleware.RequireRole(models.RoleManager, models.RoleAdmin)

	mux.HandleFunc("GET /api/health", healthHandler(d.DB))
	mux.HandleFunc("POST /api/auth/register", auth.Register)
	mux.HandleFunc("POST /api/auth/login", auth.Login)
	mux.Handle("GET /api/me", authed(http.HandlerFunc(auth.Me)))
	mux.Handle("PUT /api/me/password", authed(http.HandlerFunc(auth.ChangePassword)))
	mux.Handle("GET /api/cars", authed(http.HandlerFunc(cars.List)))
	mux.Handle("GET /api/cars/{id}", authed(http.HandlerFunc(cars.Get)))
	mux.Handle("GET /api/tariffs", authed(http.HandlerFunc(tariffs.List)))
	mux.Handle("POST /api/rentals", authed(clientOnly(http.HandlerFunc(rentals.Book))))
	mux.Handle("POST /api/rentals/{id}/start", authed(clientOnly(http.HandlerFunc(rentals.Start))))
	mux.Handle("POST /api/rentals/{id}/finish", authed(clientOnly(http.HandlerFunc(rentals.Finish))))
	mux.Handle("POST /api/rentals/{id}/cancel", authed(http.HandlerFunc(rentals.Cancel)))
	mux.Handle("GET /api/rentals/my", authed(clientOnly(http.HandlerFunc(rentals.ListMy))))
	mux.Handle("GET /api/rentals", authed(staffOnly(http.HandlerFunc(rentals.List))))
	mux.Handle("GET /api/rentals/{id}", authed(http.HandlerFunc(rentals.Get)))
	return mux
}
