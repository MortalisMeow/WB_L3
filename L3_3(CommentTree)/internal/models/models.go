package models

import "time"

type Comment struct {
	ID        int64      `json:"id"`
	ParentID  *int64     `json:"parent_id"`
	Path      string     `json:"-"`
	Body      string     `json:"body"`
	Author    string     `json:"author"`
	CreatedAt time.Time  `json:"created_at"`
	Children  []*Comment `json:"children,omitempty"`
	Level     int        `json:"level"`
	Matched   bool       `json:"matched,omitempty"`
}
