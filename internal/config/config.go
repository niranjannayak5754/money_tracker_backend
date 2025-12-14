package config

import (
	"errors"
	"os"
)

type Config struct {
	HTTPAddr  string
	MongoURI  string
	DBName    string
	JWTSecret string
}

func Load() (Config, error) {
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

	return cfg, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
