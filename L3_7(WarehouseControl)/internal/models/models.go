package models

import (
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrNotFound       = errors.New("not found")
	ErrForbidden      = errors.New("forbidden")
	ErrInvalidName    = errors.New("invalid name")
	ErrInvalidSKU     = errors.New("invalid sku")
	ErrInvalidQty     = errors.New("invalid quantity")
	ErrInvalidPrice   = errors.New("invalid price")
	ErrDuplicateSKU   = errors.New("duplicate sku")
	ErrInvalidRole    = errors.New("invalid role")
	ErrInvalidToken   = errors.New("invalid token")
)

const (
	RoleAdmin   = "admin"
	RoleManager = "manager"
	RoleViewer  = "viewer"
)

type Item struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	SKU         string    `json:"sku"`
	Quantity    int       `json:"quantity"`
	Price       float64   `json:"price"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type HistoryEntry struct {
	ID        int64           `json:"id"`
	ItemID    int64           `json:"item_id"`
	Action    string          `json:"action"`
	ChangedBy string          `json:"changed_by"`
	ChangedAt time.Time       `json:"changed_at"`
	OldData   json.RawMessage `json:"old_data,omitempty"`
	NewData   json.RawMessage `json:"new_data,omitempty"`
}

type HistoryFilter struct {
	ItemID    int64
	From      string
	To        string
	User      string
	Action    string
}

type HistoryDiff struct {
	Entry   HistoryEntry       `json:"entry"`
	Changes map[string]DiffPair  `json:"changes"`
}

type DiffPair struct {
	Old interface{} `json:"old,omitempty"`
	New interface{} `json:"new,omitempty"`
}

type CreateInput struct {
	Name        string  `json:"name"`
	SKU         string  `json:"sku"`
	Quantity    int     `json:"quantity"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
}

type UpdateInput struct {
	Name        string  `json:"name"`
	SKU         string  `json:"sku"`
	Quantity    int     `json:"quantity"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
}

type LoginInput struct {
	Username string `json:"username"`
	Role     string `json:"role"`
}

type LoginResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

type Claims struct {
	Username string
	Role     string
}
