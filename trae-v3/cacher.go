package deepseek

import (
	"context"
	"time"
)

// Store 底层存储接口
type Store interface {
	Get(ctx context.Context, key string, dst interface{}) (bool, error)
	MGet(ctx context.Context, keys []string, dstMap interface{}) error
	Exists(ctx context.Context, keys []string) (map[string]bool, error)
	MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) (int64, error)
}

// FallbackFunc 回退函数类型
type FallbackFunc func(ctx context.Context, key string) (interface{}, bool, error)

// BatchFallbackFunc 批量回退函数类型
type BatchFallbackFunc func(ctx context.Context, keys []string) (map[string]interface{}, error)

// CacheOptions 缓存选项
type CacheOptions struct {
	TTL time.Duration
}

// Cacher 高级缓存接口
type Cacher interface {
	Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error)
	MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error
	MDelete(ctx context.Context, keys []string) (int64, error)
	MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error
}