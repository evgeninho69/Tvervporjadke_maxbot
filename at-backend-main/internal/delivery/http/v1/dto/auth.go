package dto

import (
	"errors"
)

type TokenPairResponse struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
}

type ConfirmLoginRequest struct {
	Email *string `json:"email"`
	Code  *uint64 `json:"code"`
}

// Validate проверяет поле email на пустоту и соответствие паттерну,
// но игнорирует содержимое поля code.
func (clr ConfirmLoginRequest) Validate() error {
	var err error

	err = errors.Join(err, emailValidate(clr.Email))

	if clr.Code == nil {
		err = errors.Join(err, errors.New("code field is missing or null"))
	}

	return err
}

type RefreshTokenRequest struct {
	Token *string `json:"refresh_token"`
}

func (rt RefreshTokenRequest) Validate() error {
	var err error

	if rt.Token == nil {
		err = errors.Join(err, errors.New("refresh_token field is missing or null"))
	} else {
		if *rt.Token == "" {
			err = errors.Join(err, errors.New("refresh_token field is empty"))
		}
	}

	return err
}
