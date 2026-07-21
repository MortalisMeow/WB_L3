package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"shortener/internal/model"
	"strings"
)

var (
	ErrNotFound  = errors.New("record not found")
	ErrDuplicate = errors.New("duplicate record")
)

type URLRepository interface {
	CreateURL(ctx context.Context, url *model.URL) (*model.URL, error)
	GetURLByShortCode(ctx context.Context, code string) (*model.URL, error)
	RecordClick(ctx context.Context, click *model.Click) error
	GetAnalytics(ctx context.Context, code string) (*model.Analytics, error)
}

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateURL(ctx context.Context, url *model.URL) (*model.URL, error) {
	query := `
        INSERT INTO urls (original_url, short_code)
        VALUES ($1, $2)
        RETURNING id, created_at
    `

	result := &model.URL{
		OriginalURL: url.OriginalURL,
		ShortCode:   url.ShortCode,
	}

	err := r.db.QueryRowContext(ctx, query, url.OriginalURL, url.ShortCode).
		Scan(&result.ID, &result.CreatedAt)

	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("inserting URL: %w", err)
	}

	return result, nil
}

func (r *PostgresRepository) GetURLByShortCode(ctx context.Context, code string) (*model.URL, error) {
	query := `
        SELECT id, original_url, short_code, created_at
        FROM urls
        WHERE short_code = $1
    `

	url := &model.URL{}

	err := r.db.QueryRowContext(ctx, query, code).
		Scan(&url.ID, &url.OriginalURL, &url.ShortCode, &url.CreatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("getting URL by code: %w", err)
	}

	return url, nil
}

func (r *PostgresRepository) RecordClick(ctx context.Context, click *model.Click) error {
	query := `
        INSERT INTO clicks (url_id, user_agent)
        VALUES ($1, $2)
    `

	_, err := r.db.ExecContext(ctx, query, click.URLID, click.UserAgent)
	if err != nil {
		return fmt.Errorf("recording click: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetAnalytics(ctx context.Context, code string) (*model.Analytics, error) {
	url, err := r.GetURLByShortCode(ctx, code)
	if err != nil {
		return nil, err
	}

	var totalClicks int
	err = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM clicks WHERE url_id = $1`, url.ID).
		Scan(&totalClicks)
	if err != nil {
		return nil, fmt.Errorf("counting total clicks: %w", err)
	}

	userAgents, err := r.queryStrings(ctx,
		`SELECT user_agent FROM clicks WHERE url_id = $1 AND user_agent != '' ORDER BY clicked_at DESC`,
		url.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying user agents: %w", err)
	}

	clickTimes, err := r.queryStrings(ctx,
		`SELECT TO_CHAR(clicked_at, 'YYYY-MM-DD HH24:MI:SS') FROM clicks WHERE url_id = $1 ORDER BY clicked_at DESC`,
		url.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying click times: %w", err)
	}

	return &model.Analytics{
		ShortURL:    url.ShortCode,
		TotalClicks: totalClicks,
		UserAgents:  userAgents,
		ClickTimes:  clickTimes,
	}, nil
}

func (r *PostgresRepository) queryStrings(ctx context.Context, query string, args ...interface{}) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []string

	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		result = append(result, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if result == nil {
		result = []string{}
	}

	return result, nil
}

func isUniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "unique_violation")
}
