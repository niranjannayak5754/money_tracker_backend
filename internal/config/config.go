package config

import (
	"log"
	"os"
)

type Config struct {
	HTTPAddr  string
	MongoURI  string
	DBName    string
	JWTSecret string
}

func Load() Config {
	return Config{
		HTTPAddr:  env("HTTP_ADDR", ":8080"),
		MongoURI:  must("MONGO_URI"),
		DBName:    env("MONGO_DB", "money_tracker"),
		JWTSecret: must("JWT_SECRET"),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func must(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("missing required env var: %s", key)
	}
	return v
}
