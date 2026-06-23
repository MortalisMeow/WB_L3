package domain

import "time"

type Status string

const (
	StatusPending   Status = "pending"   //создано, но не запланировано
	StatusScheduled Status = "scheduled" //запланировано
	StatusSent      Status = "sent"      // отправлено
	StatusFailed    Status = "failed"    // попытка не увеначалась
	StatusCancelled Status = "cancelled" // отмена отправки
)

type Notification struct {
	ID            string     `json:"id"`
	Receiver      string     `json:"receiver"` //получатель сообщения
	Topic         string     `json:"topic"`    //текст сообщения
	ScheduledAt   time.Time  `json:"scheduled_at"`
	Status        Status     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	RetryCount    int        `json:"retry_count"`
	LastAttemptAt *time.Time `json:"last_attempt_at,omitempty"`
}

type NotificationRepo interface {
	Save(n Notification) error
	FindByID(id string) (Notification, error)
	FindPendingToSend(now time.Time) ([]Notification, error)
}

type Sender interface {
	Send(n Notification) error
}
