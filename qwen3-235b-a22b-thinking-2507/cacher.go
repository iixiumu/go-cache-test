package cache

import (
	"context"
	"time"
)

type CacheOptions struct {
	TTL time.Duration
}

type FallbackFunc func(ctx context.Context, key string) (interface{}, bool, error)

type BatchFallbackFunc func(ctx context.Context, keys []string) (map[string]interface{}, error)

type Cacher interface {
	Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error)
	MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error
	MDelete(ctx context.Context, keys []string) (int64, error)
	MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error
}