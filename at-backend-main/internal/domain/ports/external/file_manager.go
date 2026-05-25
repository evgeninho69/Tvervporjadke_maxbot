package external

import (
	"context"
	"io"
)

type FileManager interface {
	Save(ctx context.Context, filePath string, file io.Reader) error
	Delete(ctx context.Context, filePath string) error
	GetURLToFile(ctx context.Context, filePath string) (string, error)
}
