package models

import "time"

type Announcement struct {
	ID          uint64
	Title       string
	Description string
	CreatedAt   time.Time `db:"created_at"`
	Admin       uint64
}
