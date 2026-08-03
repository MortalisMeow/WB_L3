package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"salestracker/internal/models"
)

type Repo struct {
	db *sql.DB
}

func New(db *sql.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) Migrate() error {
	_, err := r.db.Exec(`
		CREATE TABLE IF NOT EXISTS items (
			id          BIGSERIAL PRIMARY KEY,
			type        VARCHAR(10) NOT NULL CHECK (type IN ('income', 'expense')),
			amount      NUMERIC(12, 2) NOT NULL CHECK (amount >= 0),
			category    VARCHAR(100) NOT NULL DEFAULT '',
			occurred_at DATE NOT NULL,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_items_occurred_at ON items (occurred_at);
		CREATE INDEX IF NOT EXISTS idx_items_type ON items (type);
		CREATE INDEX IF NOT EXISTS idx_items_category ON items (category);
	`)
	return err
}

func (r *Repo) Create(ctx context.Context, item *models.Item) (*models.Item, error) {
	query := `
		INSERT INTO items (type, amount, category, occurred_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`
	err := r.db.QueryRowContext(ctx, query, item.Type, item.Amount, item.Category, item.OccurredAt).
		Scan(&item.ID, &item.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert item: %w", err)
	}
	return item, nil
}

func (r *Repo) GetByID(ctx context.Context, id int64) (*models.Item, error) {
	item := &models.Item{}
	query := `
		SELECT id, type, amount, category, occurred_at, created_at
		FROM items WHERE id = $1
	`
	err := r.db.QueryRowContext(ctx, query, id).
		Scan(&item.ID, &item.Type, &item.Amount, &item.Category, &item.OccurredAt, &item.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("get item: %w", err)
	}
	return item, nil
}

func (r *Repo) List(ctx context.Context, f models.ItemFilter) ([]models.Item, error) {
	query := `
		SELECT id, type, amount, category, occurred_at, created_at
		FROM items
		WHERE occurred_at >= $1::date AND occurred_at <= $2::date
		  AND ($3 = '' OR type = $3)
		  AND ($4 = '' OR category = $4)
		ORDER BY occurred_at DESC, id DESC
	`
	from := defaultDate(f.From, "1970-01-01")
	to := defaultDate(f.To, "2099-12-31")

	rows, err := r.db.QueryContext(ctx, query, from, to, f.Type, f.Category)
	if err != nil {
		return nil, fmt.Errorf("list items: %w", err)
	}
	defer rows.Close()

	var list []models.Item
	for rows.Next() {
		var item models.Item
		if err := rows.Scan(&item.ID, &item.Type, &item.Amount, &item.Category, &item.OccurredAt, &item.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	if list == nil {
		list = []models.Item{}
	}
	return list, rows.Err()
}

func (r *Repo) Update(ctx context.Context, item *models.Item) error {
	query := `
		UPDATE items SET type = $1, amount = $2, category = $3, occurred_at = $4
		WHERE id = $5
	`
	res, err := r.db.ExecContext(ctx, query, item.Type, item.Amount, item.Category, item.OccurredAt, item.ID)
	if err != nil {
		return fmt.Errorf("update item: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return models.ErrNotFound
	}
	return nil
}

func (r *Repo) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM items WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete item: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return models.ErrNotFound
	}
	return nil
}

func (r *Repo) Analytics(ctx context.Context, f models.ItemFilter) (*models.Analytics, error) {
	query := `
		SELECT
			COUNT(*)::bigint,
			COALESCE(SUM(amount), 0),
			COALESCE(AVG(amount), 0),
			COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY amount), 0),
			COALESCE(PERCENTILE_CONT(0.9) WITHIN GROUP (ORDER BY amount), 0),
			COALESCE(SUM(amount) FILTER (WHERE type = 'income'), 0),
			COALESCE(SUM(amount) FILTER (WHERE type = 'expense'), 0)
		FROM items
		WHERE occurred_at >= $1::date AND occurred_at <= $2::date
		  AND ($3 = '' OR type = $3)
		  AND ($4 = '' OR category = $4)
	`
	from := defaultDate(f.From, "1970-01-01")
	to := defaultDate(f.To, "2099-12-31")

	a := &models.Analytics{}
	err := r.db.QueryRowContext(ctx, query, from, to, f.Type, f.Category).Scan(
		&a.Count, &a.Sum, &a.Avg, &a.Median, &a.Percentile90, &a.IncomeSum, &a.ExpenseSum,
	)
	if err != nil {
		return nil, fmt.Errorf("analytics: %w", err)
	}
	return a, nil
}

func (r *Repo) DailyTotals(ctx context.Context, f models.ItemFilter) ([]models.DailyTotal, error) {
	query := `
		SELECT occurred_at::text,
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE -amount END), 0)
		FROM items
		WHERE occurred_at >= $1::date AND occurred_at <= $2::date
		  AND ($3 = '' OR type = $3)
		  AND ($4 = '' OR category = $4)
		GROUP BY occurred_at
		ORDER BY occurred_at
	`
	from := defaultDate(f.From, "1970-01-01")
	to := defaultDate(f.To, "2099-12-31")

	rows, err := r.db.QueryContext(ctx, query, from, to, f.Type, f.Category)
	if err != nil {
		return nil, fmt.Errorf("daily totals: %w", err)
	}
	defer rows.Close()

	var list []models.DailyTotal
	for rows.Next() {
		var d models.DailyTotal
		if err := rows.Scan(&d.Date, &d.Total); err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	if list == nil {
		list = []models.DailyTotal{}
	}
	return list, rows.Err()
}

func defaultDate(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func ParseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}
