package apperror

import "errors"

var (
	ErrService = errors.New("service internal error")
)

// admin
var (
	ErrUserNotAdmin      = errors.New("user is not an administrator")
	ErrUserNotSuperAdmin = errors.New("user is not an super administrator")
	ErrUserSuperAdmin    = errors.New("user is super administrator")
)

// claim
var (
	ErrUnknownClaimCategory       = errors.New("unknown claim category")
	ErrIsNotImage                 = errors.New("unknown type, want \"image/png\" or \"image/jpeg\"")
	ErrPointIsNotInPolygon        = errors.New("coordinates of the claim outside the permissible district")
	ErrUserCannotPerformOperation = errors.New("user cannot perform this operation")
	ErrUnknownClaimStatus         = errors.New("unknown claim status")
)

// auth
var (
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrInvalidMailCode     = errors.New("invalid mail code")
	ErrUserNotRequestLogin = errors.New("user did not request to log in")

	ErrRefreshTokenExpired = errors.New("refresh token expired")
	ErrRefreshTokenRevoked = errors.New("refresh token has been revoked")

	ErrLoginAttemptsLimitExceeded = errors.New("login attempts limit exceeded")
)

// categories
var (
	ErrBadJSON           = errors.New("bad json")
	ErrUnprocessableJSON = errors.New("unprocessable json")
)
