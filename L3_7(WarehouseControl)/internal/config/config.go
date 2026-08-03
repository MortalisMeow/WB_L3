package config

import (
	"os"
	"time"
)

type Config struct {
	ServerPort  string
	DatabaseURL string
	JWTSecret   string
	ReadTimeout time.Duration
}

func Load() *Config {
	return &Config{
		ServerPort:  getEnv("SERVER_PORT", ":8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://warehouse:warehouse@localhost:5433/warehousecontrol?sslmode=disable"),
		JWTSecret:   getEnv("JWT_SECRET", "warehouse-secret-key"),
		ReadTimeout: 10 * time.Second,
	}
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}
