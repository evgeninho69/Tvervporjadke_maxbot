package dto

import (
	"errors"
	"time"
)

type AnnouncementResponse struct {
	ID          uint64    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type AnnouncementForAdminsResponse struct {
	AnnouncementResponse
	CreatedBy uint64 `json:"created_by"`
	UpdatedBy uint64 `json:"updated_by"`
}

type CreateAnnouncementRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
}

func (car CreateAnnouncementRequest) Validate() error {
	var err error

	if car.Title == nil {
		err = errors.Join(err, errors.New("title field is missing or null"))
	} else {
		if *car.Title == "" {
			err = errors.Join(err, errors.New("title field is empty"))
		}
	}

	if car.Description == nil {
		err = errors.Join(err, errors.New("description field is missing or null"))
	} else {
		if *car.Description == "" {
			err = errors.Join(err, errors.New("description field is empty"))
		}
	}

	return err
}

type AnnouncementUpdateRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
}

func (aur AnnouncementUpdateRequest) Validate() error {
	var err error

	if aur.Title == nil {
		err = errors.Join(err, errors.New("title field is missing or null"))
	} else {
		if *aur.Title == "" {
			err = errors.Join(err, errors.New("title field is empty"))
		}
	}

	if aur.Description == nil {
		err = errors.Join(err, errors.New("description field is missing or null"))
	} else {
		if *aur.Description == "" {
			err = errors.Join(err, errors.New("description field is empty"))
		}
	}

	return err
}
