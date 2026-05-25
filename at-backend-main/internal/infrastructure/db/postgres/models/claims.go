package models

import "activeTver/internal/domain/entity"

type ClaimWithUserEmail struct {
	entity.Claim
	UserEmail string `db:"email"`
}
