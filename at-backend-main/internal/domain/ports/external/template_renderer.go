package external

import (
	"context"
)

type TemplateRenderer interface {
	RenderLoginEmailTemplate(ctx context.Context, values any) ([]byte, error)
}
