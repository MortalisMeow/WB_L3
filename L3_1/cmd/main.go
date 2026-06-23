package main

import (
	"delayed-notifier/internal/domain"
	"delayed-notifier/internal/repo"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"delayed-notifier/internal/delivery"
	"delayed-notifier/internal/sender"
	"delayed-notifier/internal/usecase"

	"github.com/gorilla/mux"
)

func main() {
	log.Println("=== DelayedNotifier ===")

	// Хранилище статусов (Memory)
	statusRepo := repo.NewMemoryRepository()

	// Очередь (RabbitMQ)
	queueRepo, err := repo.NewRabbitMQRepository("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatalf("RabbitMQ: %v", err)
	}
	defer queueRepo.Close()

	snd := sender.NewConsoleSender()

	// UseCase (с двумя репозиториями)
	uc := usecase.NewNotificationUseCase(statusRepo, queueRepo, snd)

	// HTTP handler
	hdl := delivery.NewHandler(uc)

	if err := queueRepo.Consume(func(n domain.Notification) error {
		return uc.ProcessFromQueue(n)
	}); err != nil {
		log.Fatalf("Consumer: %v", err)
	}

	router := mux.NewRouter()
	hdl.Routes(router)
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	go func() {
		addr := ":8080"
		log.Printf("Server: http://localhost%s", addr)
		log.Println("  POST   /notify")
		log.Println("  GET    /notify/{id}")
		log.Println("  DELETE /notify/{id}")
		if err := http.ListenAndServe(addr, router); err != nil {
			log.Fatal(err)
		}
	}()

	log.Println("=== Ready ===")

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")
}
