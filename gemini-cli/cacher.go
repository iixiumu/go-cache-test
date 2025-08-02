package cache

import (
    "context"
    "time"
)

// Store is the interface for the storage backend.
type Store interface {
    Get(ctx context.Context, key string, dst interface{}) (bool, error)
    MGet(ctx context.Context, keys []string, dstMap interface{}) error
    Exists(ctx context.Context, keys []string) (map[string]bool, error)
    MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error
    Del(ctx context.Context, keys ...string) (int64, error)
}

// FallbackFunc is the function type for single key fallback.
type FallbackFunc func(ctx context.Context, key string) (interface{}, bool, error)

// BatchFallbackFunc is the function type for batch key fallback.
type BatchFallbackFunc func(ctx context.Context, keys []string) (map[string]interface{}, error)

// CacheOptions provides options for caching operations.
type CacheOptions struct {
    TTL time.Duration
}

// Cacher is the interface for the advanced cache.
type Cacher interface {
    Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error)
    MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error
    MDelete(ctx context.Context, keys []string) (int64, error)
    MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error
}
