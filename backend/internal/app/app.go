package app

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"carsharing/internal/handlers"
	"carsharing/internal/repository"
	"carsharing/internal/services"
	"carsharing/internal/util"
	"carsharing/internal/web"
)

func NewHandler(pool *pgxpool.Pool, tokens *util.TokenManager, frontendDir string) (http.Handler, error) {
	site, err := web.New(frontendDir)
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.Handle("/api/", newAPI(pool, tokens))
	mux.Handle("/", site)
	return web.SecurityHeaders(mux), nil
}

func newAPI(pool *pgxpool.Pool, tokens *util.TokenManager) http.Handler {
	users := repository.NewUserRepository(pool)
	cars := repository.NewCarRepository(pool)
	return handlers.NewRouter(handlers.Deps{
		DB:      pool,
		Auth:    services.NewAuthService(users, tokens),
		Tokens:  tokens,
		Users:   users,
		Cars:    services.NewCarService(cars),
		Tariffs: services.NewTariffService(repository.NewTariffRepository(pool)),
		Rentals: services.NewRentalService(repository.NewRentalRepository(pool)),
	})
}

func RunBookingExpiry(ctx context.Context, pool *pgxpool.Pool, every time.Duration) {
	svc := services.NewRentalService(repository.NewRentalRepository(pool))
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		if n, err := svc.ExpireBookings(ctx); err != nil && ctx.Err() == nil {
			log.Printf("booking expiry: %v", err)
		} else if n > 0 {
			log.Printf("booking expiry: отменено просроченных бронирований: %d", n)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
