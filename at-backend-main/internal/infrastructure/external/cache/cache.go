package cache

import (
	"activeTver/internal/pkg/apperror"
	"activeTver/internal/pkg/common"
	"activeTver/internal/pkg/logs"
	"context"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

type cacheStorage[Key comparable, Value any] struct {
	lru        *expirable.LRU[Key, Value]
	dummyValue Value
}

func NewCacheStorage[Key comparable, Value any](expirationTime time.Duration) *cacheStorage[Key, Value] {
	return &cacheStorage[Key, Value]{
		lru: expirable.NewLRU[Key, Value](
			10000,
			nil,
			expirationTime,
		),
		dummyValue: *new(Value),
	}
}

const opAdd = "infrastructure.external.CacheStorage.Add"

func (cs *cacheStorage[Key, Value]) Add(ctx context.Context, key Key, value Value, expire time.Time) error {
	if ok := cs.lru.Add(key, value); ok {
		logs.Warn(
			"eviction",
			common.GetCorrelationID(ctx),
			opAdd,
		)
	}

	return nil
}

const opRemove = "infrastructure.external.CacheStorage.Remove"

func (cs *cacheStorage[Key, Value]) Remove(ctx context.Context, key Key) error {
	if ok := cs.lru.Remove(key); !ok {
		logs.Warn(
			apperror.ErrValueWasNotCached.Error(),
			common.GetCorrelationID(ctx),
			opRemove,
		)

		return apperror.ErrValueWasNotCached
	}

	return nil
}

const opGet = "infrastructure.external.CacheStorage.Get"

func (cs *cacheStorage[Key, Value]) Get(ctx context.Context, key Key) (Value, error) {
	value, ok := cs.lru.Get(key)
	if !ok {
		logs.Warn(
			apperror.ErrValueWasNotCached.Error(),
			common.GetCorrelationID(ctx),
			opGet,
		)

		return cs.dummyValue, apperror.ErrValueWasNotCached
	}

	return value, nil
}

const opContains = "infrastructure.external.CacheStorage.Contains"

func (cs *cacheStorage[Key, Value]) Contains(ctx context.Context, key Key) (bool, error) {
	if ok := cs.lru.Contains(key); !ok {
		logs.Warn(
			apperror.ErrValueWasNotCached.Error(),
			common.GetCorrelationID(ctx),
			opContains,
		)

		return false, apperror.ErrValueWasNotCached
	}

	return true, nil
}
