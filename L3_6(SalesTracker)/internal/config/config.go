package config

import (
	"os"
	"time"
)

type Config struct {
	ServerPort  string
	DatabaseURL string
	ReadTimeout time.Duration
}

func Load() *Config {
	return &Config{
		ServerPort:  getEnv("SERVER_PORT", ":8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://sales:sales@localhost:5432/salestracker?sslmode=disable"),
		ReadTimeout: 10 * time.Second,
	}
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}
