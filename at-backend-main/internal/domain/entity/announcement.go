package entity

import "time"

type Announcement struct {
	ID          uint64
	Title       string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CreatedBy   uint64
	UpdatedBy   uint64
}
