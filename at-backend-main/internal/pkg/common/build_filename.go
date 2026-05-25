package common

import (
	"github.com/google/uuid"
)

func BuildFileName(filetype string) string {
    return uuid.NewString() + filetype
}