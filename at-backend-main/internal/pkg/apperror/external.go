package apperror

import "errors"

// email sender
var (
	ErrEmailSender = errors.New("email sender internal error")
)

// file manager
var (
	ErrFileManager   = errors.New("file manager internal error")
	ErrFileNotExists = errors.New("file or directory does not exists")
	ErrFileExists    = errors.New("file exists")
)

// template renderer
var (
	ErrTemplateRenderer = errors.New("template renderer internal error")
)

// cache storage
var (
	ErrCacheStorage       = errors.New("cache storage internal error")
	ErrValueAlreadyCached = errors.New("value is already cached")
	ErrValueWasNotCached  = errors.New("value was not cached")
)
