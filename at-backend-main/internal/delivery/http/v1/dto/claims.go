package dto

import (
	"activeTver/internal/domain/consts"
	"activeTver/internal/domain/types"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"
)

type ClaimResponse struct {
	ID                uint64            `json:"id"`
	Title             string            `json:"title" validate:"required"`
	Description       string            `json:"description" validate:"required"`
	Category          string            `json:"category" validate:"required"`
	Status            types.ClaimStatus `json:"status"`
	Photos            []string          `json:"photos,omitempty"`
	Latitude          float64           `json:"latitude" validate:"latitude,required"`
	Longitude         float64           `json:"longitude" validate:"longitude,required"`
	Feedback          string            `json:"feedback"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
	StatusUpdatedAt   time.Time         `json:"status_updated_at"`
	FeedbackUpdatedAt time.Time         `json:"feedback_updated_at"`
}

type ClaimForAdminsResponse struct {
	ClaimResponse
	Email     string `json:"email"`
	CreatedBy uint64 `json:"created_by"`
}

type ClaimForPaginationResponse struct {
	ID          uint64            `json:"id"`
	Title       string            `json:"title" validate:"required"`
	Description string            `json:"description" validate:"required"`
	Category    string            `json:"category" validate:"required"`
	Status      types.ClaimStatus `json:"status"`
	Latitude    float64           `json:"latitude" validate:"latitude,required"`
	Longitude   float64           `json:"longitude" validate:"longitude,required"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type ClaimIDResponse struct {
	ID uint64 `json:"id"`
}

type ClaimsDataForMapResponse struct {
	ID        uint64  `json:"id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type ClaimFeedbackRequest struct {
	Feedback *string `json:"feedback"`
}

func (claim ClaimFeedbackRequest) Validate() error {
	var err error

	if claim.Feedback == nil {
		err = errors.Join(err, errors.New("feedback field is missing or null"))
	} else {
		if *claim.Feedback == "" {
			err = errors.Join(err, errors.New("feedback field is empty"))
		}
	}

	return err
}

type ClaimStatusUpdateRequest struct {
	Status *types.ClaimStatus `json:"status"`
}

func (claim ClaimStatusUpdateRequest) Validate() error {
	var err error

	if claim.Status == nil {
		err = errors.Join(err, errors.New("status field is missing or null"))
	} else {
		if *claim.Status == "" {
			err = errors.Join(err, errors.New("status field is empty"))
		}
	}

	return err
}

type ClaimWithoutCategory struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

func (claim ClaimWithoutCategory) Validate() error {
	var err error

	if claim.Title == "" {
		err = errors.Join(err, errors.New("title field is missing or empty"))
	}

	if utf8.RuneCountInString(claim.Title) > consts.ClaimTitleCharactersMaxCount {
		err = errors.Join(err, fmt.Errorf("title field value is too long; it can be <= %d characters", consts.ClaimTitleCharactersMaxCount))
	}

	if claim.Description == "" {
		err = errors.Join(err, errors.New("description field is missing or empty"))
	}

	if utf8.RuneCountInString(claim.Description) > consts.ClaimDescriptionCharactersMaxCount {
		err = errors.Join(err, fmt.Errorf("description field value is too long; it can be <= %d characters", consts.ClaimDescriptionCharactersMaxCount))
	}

	if !(-180 <= claim.Longitude || claim.Longitude <= 180) {
		err = errors.Join(err, errors.New("longitude field has an invalid value or missing"))
	}

	if !(-90 <= claim.Latitude || claim.Latitude <= 90) {
		err = errors.Join(err, errors.New("latitude field has an invalid value or missing"))
	}

	return err
}

type UpdateClaimRequest struct {
	ClaimWithoutCategory
}

func (claim UpdateClaimRequest) Validate() error {
	var err error

	err = errors.Join(err, claim.ClaimWithoutCategory.Validate())

	return err
}

type CreateProposalRequest struct {
	ClaimWithoutCategory
}

func (claim CreateProposalRequest) Validate() error {
	var err error

	err = errors.Join(err, claim.ClaimWithoutCategory.Validate())

	return err
}

type CreateProblemRequest struct {
	ClaimWithoutCategory
	Category string `json:"category"`
}

func (claim CreateProblemRequest) Validate() error {
	var err error

	err = errors.Join(err, claim.ClaimWithoutCategory.Validate())

	if claim.Category == "" {
		err = errors.Join(err, errors.New("category field is missing or empty"))
	}

	return err
}
