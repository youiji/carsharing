package config

import (
	"errors"
	"fmt"
	"os"
	"time"
)

type Config struct {
	Addr        string
	DatabaseURL string
	JWTSecret   string
	JWTTTL      time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		Addr:        getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		JWTTTL:      30 * time.Minute,
	}

	if v := os.Getenv("JWT_TTL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("JWT_TTL: %w", err)
		}
		cfg.JWTTTL = d
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL не задана")
	}
	if len(cfg.JWTSecret) < 32 {
		return Config{}, errors.New("JWT_SECRET должна быть не короче 32 символов")
	}
	if cfg.JWTTTL <= 0 {
		return Config{}, errors.New("JWT_TTL должен быть больше нуля")
	}
	return cfg, nil
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}
