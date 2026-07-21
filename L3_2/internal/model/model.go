package model

import "time"

type URL struct {
	ID          int64     `json:"id"`
	OriginalURL string    `json:"original_url"`
	ShortCode   string    `json:"short_code"`
	CreatedAt   time.Time `json:"created_at"`
}

type Click struct {
	ID        int64     `json:"id"`
	URLID     int64     `json:"url_id"`
	ClickedAt time.Time `json:"clicked_at"`
	UserAgent string    `json:"user_agent,omitempty"`
}

type Analytics struct {
	ShortURL    string   `json:"short_url"`
	TotalClicks int      `json:"total_clicks"`
	UserAgents  []string `json:"user_agents"`
	ClickTimes  []string `json:"click_times"`
}
