package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPAddr       string
	MongoURI       string
	DBName         string
	JWTSecret      string
	AllowedOrigins []string
	AppTimezone    *time.Location
}

func Load() (Config, error) {
	// Loads .env into the process environment if present; does not override
	// vars already set (e.g. in production, where .env doesn't exist).
	_ = godotenv.Load()

	cfg := Config{
		HTTPAddr:  env("HTTP_ADDR", ":8080"),
		MongoURI:  os.Getenv("MONGO_URI"),
		DBName:    env("MONGO_DB", "money_tracker"),
		JWTSecret: os.Getenv("JWT_SECRET"),
	}

	if cfg.MongoURI == "" {
		return Config{}, errors.New("MONGO_URI is required")
	}
	if cfg.JWTSecret == "" {
		return Config{}, errors.New("JWT_SECRET is required")
	}

	for _, o := range strings.Split(env("ALLOWED_ORIGINS", "http://localhost:3000"), ",") {
		if o = strings.TrimSpace(o); o != "" {
			cfg.AllowedOrigins = append(cfg.AllowedOrigins, o)
		}
	}

	tzName := env("APP_TIMEZONE", "Asia/Kolkata")
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return Config{}, fmt.Errorf("invalid APP_TIMEZONE %q: %w", tzName, err)
	}
	cfg.AppTimezone = loc

	return cfg, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
