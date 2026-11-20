package app

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"carsharing/internal/handlers"
	"carsharing/internal/repository"
	"carsharing/internal/services"
)

func NewHandler(pool *pgxpool.Pool) http.Handler {
	users := repository.NewUserRepository(pool)
	return handlers.NewRouter(handlers.Deps{
		DB:   pool,
		Auth: services.NewAuthService(users),
	})
}
