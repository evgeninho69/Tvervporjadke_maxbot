package entity

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID     uuid.UUID
	UserID uint64
	Token  string
	Hash   string
	Salt   string
	Expire time.Time
	Revoked bool
}
