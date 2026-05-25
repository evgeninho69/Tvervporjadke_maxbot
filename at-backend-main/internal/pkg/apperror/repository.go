package apperror

import "errors"

var (
	ErrUseCase = errors.New("usecase layer internal error")
)

var (
	ErrRepository = errors.New("repository internal error")
)

// announcement
var (
	ErrAnnouncementNotExists = errors.New("announcement is not found")
)

// claims
var (
	ErrClaimNotExists = errors.New("claim does not exists")
)

// users
var (
	ErrUserNotBlocked     = errors.New("user is not blocked")
	ErrUserAlreadyBlocked = errors.New("user is already blocked")

	ErrUserExists    = errors.New("user already exists")
	ErrUserNotExists = errors.New("user does not exists")
)

// tokens
var (
	ErrTokenNotExists = errors.New("token does not exists")
	ErrTokenExists    = errors.New("token already exists")
)

// admin
var (
	ErrUserAlreadyAdmin = errors.New("user is already admin")
)
