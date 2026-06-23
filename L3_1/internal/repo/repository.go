package repo

import (
	"errors"
	"sync"
	"time"

	"delayed-notifier/internal/domain"
)

type MemoryRepository struct {
	mu            sync.RWMutex
	notifications map[string]domain.Notification
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		notifications: make(map[string]domain.Notification),
	}
}

func (r *MemoryRepository) Save(n domain.Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.notifications[n.ID] = n
	return nil
}

func (r *MemoryRepository) FindByID(id string) (domain.Notification, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	n, exists := r.notifications[id]
	if !exists {
		return domain.Notification{}, errors.New("notification not found")
	}
	return n, nil
}

func (r *MemoryRepository) FindPendingToSend(now time.Time) ([]domain.Notification, error) {
	return nil, errors.New("not used with RabbitMQ")
}
