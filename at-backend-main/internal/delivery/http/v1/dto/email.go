package dto

import (
	"errors"
	"regexp"
)

var emailRegex = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func emailValidate(email *string) error {
	var err error

	if email == nil {
		err = errors.Join(err, errors.New("email field is missing or null"))
	} else {
		if *email == "" {
			err = errors.Join(err, errors.New("email field is empty"))
		}

		if !emailRegex.MatchString(*email) && *email != "" {
			err = errors.Join(err, errors.New("email field contains an incorrect value"))
		}
	}

	return err
}

type EmailResponseAndRequest struct {
	Email *string `json:"email"`
}

func (e EmailResponseAndRequest) Validate() error {
	var err error

	err = errors.Join(err, emailValidate(e.Email))

	return err
}
