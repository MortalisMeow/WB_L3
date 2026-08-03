package service

import (
	"context"
	"strings"
	"time"

	"salestracker/internal/models"
	"salestracker/internal/repo"
)

type Service struct {
	repo *repo.Repo
}

func New(r *repo.Repo) *Service {
	return &Service{repo: r}
}

func (s *Service) Create(ctx context.Context, in models.CreateInput) (*models.Item, error) {
	item, err := s.validate(in.Type, in.Amount, in.Category, in.OccurredAt)
	if err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, item)
}

func (s *Service) Get(ctx context.Context, id int64) (*models.Item, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context, f models.ItemFilter) ([]models.Item, error) {
	return s.repo.List(ctx, f)
}

func (s *Service) Update(ctx context.Context, id int64, in models.UpdateInput) (*models.Item, error) {
	item, err := s.validate(in.Type, in.Amount, in.Category, in.OccurredAt)
	if err != nil {
		return nil, err
	}
	item.ID = id
	if err := s.repo.Update(ctx, item); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) Analytics(ctx context.Context, f models.ItemFilter) (*models.Analytics, error) {
	return s.repo.Analytics(ctx, f)
}

func (s *Service) DailyTotals(ctx context.Context, f models.ItemFilter) ([]models.DailyTotal, error) {
	return s.repo.DailyTotals(ctx, f)
}

func (s *Service) validate(itemType string, amount float64, category, dateStr string) (*models.Item, error) {
	itemType = strings.TrimSpace(itemType)
	if itemType != "income" && itemType != "expense" {
		return nil, models.ErrInvalidType
	}
	if amount <= 0 {
		return nil, models.ErrInvalidAmount
	}
	category = strings.TrimSpace(category)
	if category == "" {
		return nil, models.ErrEmptyCategory
	}
	d, err := repo.ParseDate(strings.TrimSpace(dateStr))
	if err != nil {
		return nil, models.ErrInvalidDate
	}
	if d.After(time.Now().AddDate(1, 0, 0)) {
		return nil, models.ErrInvalidDate
	}
	return &models.Item{
		Type:       itemType,
		Amount:     amount,
		Category:   category,
		OccurredAt: d,
	}, nil
}
