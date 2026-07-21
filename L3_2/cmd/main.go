package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"shortener/config"
	"shortener/internal/handler"
	"shortener/internal/service"

	_ "github.com/lib/pq"
)

func main() {
	cfg := config.Load()

	// Подключаемся к бд
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Connected to database")

	// Создаём слои
	svc := service.NewURLShortenerService(db, cfg.BaseURL)
	h := handler.NewHandler(svc)

	// Настраиваем роутер
	mux := http.NewServeMux()
	mux.HandleFunc("/shorten", h.ShortenURL)
	mux.HandleFunc("/s/", h.Redirect)
	mux.HandleFunc("/analytics/", h.Analytics)

	server := &http.Server{
		Addr:         cfg.ServerPort,
		Handler:      mux,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	go func() {
		log.Printf("Server starting on %s", cfg.ServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
