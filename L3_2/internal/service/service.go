package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"shortener/internal/model"
	"strings"
)

type URLService interface {
	ShortenURL(ctx context.Context, originalURL string) (string, error)
	GetOriginalURL(ctx context.Context, shortCode string) (string, error)
	GetAnalytics(ctx context.Context, shortCode string) (*model.Analytics, error)
}

const (
	shortCodeLen = 7
	maxGenAtt    = 5
	base62Chars  = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

var (
	ErrInvalidURL = errors.New("invalid URL")
	ErrCodeGen    = errors.New("error while generation code")
	ErrNotFound   = errors.New("short URL not found")
)

type URLShortenerService struct {
	db      *sql.DB
	baseURL string
}

func NewURLShortenerService(db *sql.DB, baseURL string) *URLShortenerService {
	return &URLShortenerService{
		db:      db,
		baseURL: baseURL,
	}
}

func (s *URLShortenerService) ShortenURL(ctx context.Context, originalURL string) (string, error) {
	//валидация
	if !strings.HasPrefix(originalURL, "http://") && !strings.HasPrefix(originalURL, "https://") {
		return "", ErrInvalidURL
	}

	//генерация кода
	var shortCode string
	for i := 0; i < maxGenAtt; i++ {
		shortCode, _ = generateShortCode(shortCodeLen)

		exists, err := isCodeUnique(ctx, s.db, shortCode)
		if err != nil {
			return "", fmt.Errorf("chek is code unique")
		}

		if !exists {
			break
		}
	}

	if shortCode == "" {
		return "", ErrCodeGen
	}

	//сохраняем в бд
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO urls (original_url, short_code) VALUES ($1, $2)",
		originalURL, shortCode,
	)
	if err != nil {
		return "", fmt.Errorf("saving URL: %w", err)
	}

	shortURL := fmt.Sprintf("%s/s/%s", s.baseURL, shortCode)

	return shortURL, nil
}

func (s *URLShortenerService) GetOriginalURL(ctx context.Context, shortCode string) (string, error) {
	var url model.URL
	err := s.db.QueryRowContext(ctx,
		"SELECT id, original_url, short_code, created_at FROM urls WHERE short_code = $1",
		shortCode,
	).Scan(&url.ID, &url.OriginalURL, &url.ShortCode, &url.CreatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("getting URL: %w", err)
	}

	go func() {
		s.db.ExecContext(context.Background(),
			"INSERT INTO clicks (url_id, user_agent) VALUES ($1, $2)",
			url.ID, "todo",
		)
	}()

	return url.OriginalURL, nil
}

func (s *URLShortenerService) GetAnalytics(ctx context.Context, shortCode string) (*model.Analytics, error) {
	var url model.URL
	err := s.db.QueryRowContext(ctx,
		"SELECT id, original_url, short_code, created_at FROM urls WHERE short_code = $1",
		shortCode,
	).Scan(&url.ID, &url.OriginalURL, &url.ShortCode, &url.CreatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("getting URL: %w", err)
	}

	var totalClicks int
	err = s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM clicks WHERE url_id = $1", url.ID,
	).Scan(&totalClicks)
	if err != nil {
		return nil, fmt.Errorf("counting clicks: %w", err)
	}

	userAgents := s.queryStrings(ctx,
		"SELECT user_agent FROM clicks WHERE url_id = $1 AND user_agent != '' ORDER BY clicked_at DESC",
		url.ID,
	)

	clickTimes := s.queryStrings(ctx,
		"SELECT TO_CHAR(clicked_at, 'YYYY-MM-DD HH24:MI:SS') FROM clicks WHERE url_id = $1 ORDER BY clicked_at DESC",
		url.ID,
	)

	return &model.Analytics{
		ShortURL:    url.ShortCode,
		TotalClicks: totalClicks,
		UserAgents:  userAgents,
		ClickTimes:  clickTimes,
	}, nil
}

func (s *URLShortenerService) queryStrings(ctx context.Context, query string, args ...interface{}) []string {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return []string{}
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var str string
		if err := rows.Scan(&str); err != nil {
			return []string{}
		}
		result = append(result, str)
	}

	if result == nil {
		result = []string{}
	}

	return result
}

func generateShortCode(length int) (string, error) {
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(base62Chars))))
		if err != nil {
			return "", err
		}
		result[i] = base62Chars[randomIndex.Int64()]
	}
	return string(result), nil
}

func isCodeUnique(ctx context.Context, db *sql.DB, code string) (bool, error) {
	var exists bool
	err := db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM urls WHERE short_code = $1)",
		code,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("checking code uniqueness: %w", err)
	}
	return exists, nil
}
