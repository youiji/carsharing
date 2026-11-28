package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	router, err := app.NewHandler(pool, tokens, cfg.FrontendDir)
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go app.RunBookingExpiry(ctx, pool, 15*time.Second)

	srv := &http.Server{Addr: cfg.Addr, Handler: router}
	go func() {
		log.Println("запуск сервера на", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	log.Println("остановка сервера...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
