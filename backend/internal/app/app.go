package app

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"carsharing/internal/handlers"
	"carsharing/internal/repository"
	"carsharing/internal/services"
	"carsharing/internal/util"
)

func NewHandler(pool *pgxpool.Pool, tokens *util.TokenManager) http.Handler {
	users := repository.NewUserRepository(pool)
	cars := repository.NewCarRepository(pool)
	return handlers.NewRouter(handlers.Deps{
		DB:     pool,
		Auth:   services.NewAuthService(users, tokens),
		Tokens: tokens,
		Users:  users,
		Cars:   services.NewCarService(cars),
	})
}
