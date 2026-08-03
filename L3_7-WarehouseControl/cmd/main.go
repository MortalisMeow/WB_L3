package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"warehousecontrol/internal/app"
	"warehousecontrol/internal/config"
)

func main() {
	cfg := config.Load()
	application := app.New(cfg)

	go func() {
		if err := application.Run(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("run error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	application.Shutdown(ctx)
}
