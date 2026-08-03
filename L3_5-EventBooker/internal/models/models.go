package models

import "time"

type Image struct {
	ID        int64     `json:"id"`
	Filename  string    `json:"filename"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
