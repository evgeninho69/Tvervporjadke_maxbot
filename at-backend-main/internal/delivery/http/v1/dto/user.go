package dto

import (
	"errors"
	"time"
)

type UserIDResponse struct {
	UID uint64 `json:"uid" validate:"required"`
}

type UserResponse struct {
	UserIDResponse
	EmailResponseAndRequest
	RegisteredAt time.Time `json:"registered_at"`
}

type BlockDataResponse struct {
	BlockedBy uint64    `json:"blocked_by"`
	BlockedAt time.Time `json:"blocked_at"`
	Reason    string    `json:"reason"`
}

type BlockDataForPaginationResponse struct {
	ID        uint64    `json:"id"`
	BlockedAt time.Time `json:"blocked_at"`
	EmailResponseAndRequest
	UserIDResponse
}

type BlockRequest struct {
	Reason *string `json:"reason"`
}

func (s BlockRequest) Validate() error {
	var err error

	if s.Reason == nil {
		err = errors.Join(err, errors.New("reason field is missing or null"))
	} else {
		if *s.Reason == "" {
			err = errors.Join(err, errors.New("reason field is empty"))
		}
	}

	return err
}
