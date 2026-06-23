package sender

import (
	"fmt"
	"time"

	"delayed-notifier/internal/domain"
)

type ConsoleSender struct{}

func NewConsoleSender() *ConsoleSender {
	return &ConsoleSender{}
}

func (s *ConsoleSender) Send(n domain.Notification) error {
	fmt.Printf("ID:      %s\n", n.ID)
	fmt.Printf("To:      %s\n", n.Receiver)
	fmt.Printf("Message: %s\n", n.Topic)
	fmt.Printf("Time:    %s\n", n.ScheduledAt.Format(time.RFC3339))
	fmt.Printf("Retry:   %d\n", n.RetryCount)
	fmt.Printf("─────────────────────────────────────────────\n")
	return nil
}
