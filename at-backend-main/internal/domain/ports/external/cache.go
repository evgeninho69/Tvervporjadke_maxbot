package external

import (
	"context"
	"time"
)

type CacheStorage[Key comparable, Value any] interface {
	Add(ctx context.Context, key Key, value Value, expire time.Time) error
	Remove(ctx context.Context, key Key) error
	Get(ctx context.Context, key Key) (Value, error)
	Contains(ctx context.Context, key Key) (bool, error)
}