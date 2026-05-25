package entity

import (
	"activeTver/internal/domain/types"
	"time"
)

type Claim struct {
	ID                uint64
	CreatedBy         uint64
	Title             string
	Description       string
	Category          string
	Status            types.ClaimStatus
	Photos            []string
	Latitude          float64
	Longitude         float64
	Feedback          string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	StatusUpdatedAt   time.Time
	FeedbackUpdatedAt time.Time
}
