package main

import (
	"context"
	"log"
	"net/http"

	"carsharing/internal/config"
	"carsharing/internal/handlers"
	"carsharing/internal/repository"
	"carsharing/internal/services"
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
	router := handlers.NewRouter(handlers.Deps{
		DB:   pool,
		Auth: services.NewAuthService(users),
	})

	log.Println("сервер запущен на", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, router); err != nil {
		log.Fatal(err)
	}
}
