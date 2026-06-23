package usecase

import (
	"delayed-notifier/internal/domain"
	"delayed-notifier/internal/repo"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"log"
	"time"
)

type CreateNotificationRequest struct {
	Receiver    string
	Topic       string
	ScheduledAt time.Time
}

type NotificationUseCase struct {
	statusRepo domain.NotificationRepo
	queueRepo  *repo.RabbitMQRepository
	sender     domain.Sender
}

func NewNotificationUseCase(
	statusRepo domain.NotificationRepo,
	queueRepo *repo.RabbitMQRepository,
	sender domain.Sender,
) *NotificationUseCase {
	return &NotificationUseCase{
		statusRepo: statusRepo,
		queueRepo:  queueRepo,
		sender:     sender,
	}
}

func (uc *NotificationUseCase) CreateNotification(req CreateNotificationRequest) (domain.Notification, error) {
	// Валидация
	if req.ScheduledAt.Before(time.Now()) {
		return domain.Notification{}, errors.New("cannot schedule in the past")
	}
	if req.ScheduledAt.After(time.Now().Add(30 * 24 * time.Hour)) {
		return domain.Notification{}, errors.New("cannot schedule more than 30 days ahead")
	}
	if req.Topic == "" {
		return domain.Notification{}, errors.New("message cannot be empty")
	}
	if req.Receiver == "" {
		return domain.Notification{}, errors.New("recipient cannot be empty")
	}

	now := time.Now()

	n := domain.Notification{
		ID:          uuid.New().String(),
		Receiver:    req.Receiver,
		Topic:       req.Topic,
		ScheduledAt: req.ScheduledAt,
		Status:      domain.StatusScheduled,
		CreatedAt:   now,
		RetryCount:  0,
	}

	if err := uc.statusRepo.Save(n); err != nil {
		return domain.Notification{}, fmt.Errorf("status save: %w", err)
	}

	delay := req.ScheduledAt.Sub(now)
	if delay < 0 {
		delay = 0
	}
	if err := uc.queueRepo.Publish(n, delay); err != nil {
		return domain.Notification{}, fmt.Errorf("queue publish: %w", err)
	}

	log.Printf("[USECASE] Created %s (delay: %s)", n.ID, delay)
	return n, nil
}
func (uc *NotificationUseCase) GetNotification(id string) (domain.Notification, error) {
	return uc.statusRepo.FindByID(id)
}

func (uc *NotificationUseCase) CancelNotification(id string) error {
	n, err := uc.statusRepo.FindByID(id)
	if err != nil {
		return fmt.Errorf("not found: %w", err)
	}
	if n.Status == domain.StatusSent {
		return errors.New("already sent")
	}
	if n.Status == domain.StatusCancelled {
		return nil
	}
	n.Status = domain.StatusCancelled
	return uc.statusRepo.Save(n)
}

func (uc *NotificationUseCase) ProcessFromQueue(n domain.Notification) error {
	if existing, err := uc.statusRepo.FindByID(n.ID); err == nil {
		if existing.Status == domain.StatusCancelled {
			return nil
		}
	}

	err := uc.sender.Send(n)

	n.RetryCount++
	now := time.Now()
	n.LastAttemptAt = &now

	if err == nil {
		n.Status = domain.StatusSent
		uc.statusRepo.Save(n)
		log.Printf("[USECASE] Sent %s", n.ID)
		return nil
	}

	n.Status = domain.StatusFailed
	uc.statusRepo.Save(n)

	delay := time.Duration(1<<n.RetryCount) * time.Minute
	if delay > 1*time.Hour {
		delay = 1 * time.Hour
	}

	log.Printf("[USECASE] Failed %s, retry in %s", n.ID, delay)

	if pubErr := uc.queueRepo.Publish(n, delay); pubErr != nil {
		return fmt.Errorf("republish: %w", pubErr)
	}

	return err
}
