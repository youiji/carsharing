package handlers

import (
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"carsharing/internal/middleware"
	"carsharing/internal/models"
	"carsharing/internal/services"
	"carsharing/internal/util"
)

type Deps struct {
	DB          *pgxpool.Pool
	Tokens      *util.TokenManager
	Users       middleware.UserLoader
	Auth        *services.AuthService
	Cars        *services.CarService
	Tariffs     *services.TariffService
	Rentals     *services.RentalService
	Maintenance *services.MaintenanceService
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
	maintenance := &MaintenanceHandler{svc: d.Maintenance}

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
	mux.Handle("POST /api/cars", authed(staffOnly(http.HandlerFunc(cars.Create))))
	mux.Handle("PUT /api/cars/{id}", authed(staffOnly(http.HandlerFunc(cars.Update))))
	mux.Handle("PATCH /api/cars/{id}/status", authed(staffOnly(http.HandlerFunc(cars.ChangeStatus))))
	mux.Handle("DELETE /api/cars/{id}", authed(staffOnly(http.HandlerFunc(cars.Delete))))
	mux.Handle("POST /api/tariffs", authed(staffOnly(http.HandlerFunc(tariffs.Create))))
	mux.Handle("PUT /api/tariffs/{id}", authed(staffOnly(http.HandlerFunc(tariffs.Update))))
	mux.Handle("DELETE /api/tariffs/{id}", authed(staffOnly(http.HandlerFunc(tariffs.Delete))))
	mux.Handle("GET /api/maintenance", authed(staffOnly(http.HandlerFunc(maintenance.List))))
	mux.Handle("POST /api/maintenance", authed(staffOnly(http.HandlerFunc(maintenance.Create))))
	mux.Handle("PATCH /api/maintenance/{id}", authed(staffOnly(http.HandlerFunc(maintenance.Update))))
	mux.Handle("DELETE /api/maintenance/{id}", authed(staffOnly(http.HandlerFunc(maintenance.Delete))))

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		if allowed := allowedMethods(mux, r); len(allowed) > 0 {
			w.Header().Set("Allow", strings.Join(allowed, ", "))
			util.WriteError(w, r, util.ErrMethodNotAllowed)
			return
		}
		util.WriteError(w, r, util.ErrNotFound.WithMessage("Путь API не найден"))
	})
	return mux
}

func allowedMethods(mux *http.ServeMux, r *http.Request) []string {
	var allowed []string
	for _, m := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		if m == r.Method {
			continue
		}
		probe := r.Clone(r.Context())
		probe.Method = m
		if _, pattern := mux.Handler(probe); pattern != "" && pattern != "/api/" {
			allowed = append(allowed, m)
		}
	}
	return allowed
}
