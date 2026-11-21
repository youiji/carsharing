package main

import (
	"context"
	"log"
	"net/http"

	"carsharing/internal/app"
	"carsharing/internal/config"
	"carsharing/internal/repository"
	"carsharing/internal/util"
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

	tokens := util.NewTokenManager(cfg.JWTSecret, cfg.JWTTTL)
	router := app.NewHandler(pool, tokens)

	log.Println("сервер запущен на", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, router); err != nil {
		log.Fatal(err)
	}

}
