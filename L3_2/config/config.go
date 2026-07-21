package config

import (
	"os"
	"time"
)

type Config struct {
	ServerPort   string
	DatabaseURL  string
	BaseURL      string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func Load() *Config {
	return &Config{
		ServerPort:   getEnv("SERVER_PORT", ":8080"),
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://user:pass@localhost:5432/shortener?sslmode=disable"),
		BaseURL:      getEnv("BASE_URL", "http://localhost:8080"),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
