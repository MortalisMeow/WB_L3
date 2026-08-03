package models

import (
	"errors"
	"time"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrInvalidType   = errors.New("invalid type")
	ErrInvalidAmount = errors.New("invalid amount")
	ErrInvalidDate   = errors.New("invalid date")
	ErrEmptyCategory = errors.New("empty category")
)

type Item struct {
	ID         int64     `json:"id"`
	Type       string    `json:"type"`
	Amount     float64   `json:"amount"`
	Category   string    `json:"category"`
	OccurredAt time.Time `json:"occurred_at"`
	CreatedAt  time.Time `json:"created_at"`
}

type ItemFilter struct {
	From     string
	To       string
	Type     string
	Category string
}

type Analytics struct {
	Count        int64   `json:"count"`
	Sum          float64 `json:"sum"`
	Avg          float64 `json:"avg"`
	Median       float64 `json:"median"`
	Percentile90 float64 `json:"percentile_90"`
	IncomeSum    float64 `json:"income_sum"`
	ExpenseSum   float64 `json:"expense_sum"`
}

type DailyTotal struct {
	Date  string  `json:"date"`
	Total float64 `json:"total"`
}

type CreateInput struct {
	Type       string  `json:"type"`
	Amount     float64 `json:"amount"`
	Category   string  `json:"category"`
	OccurredAt string  `json:"occurred_at"`
}

type UpdateInput struct {
	Type       string  `json:"type"`
	Amount     float64 `json:"amount"`
	Category   string  `json:"category"`
	OccurredAt string  `json:"occurred_at"`
}
