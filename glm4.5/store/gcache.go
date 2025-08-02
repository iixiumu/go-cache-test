package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/bluele/gcache"
	"go-cache/internal"
)

// GCacheStore GCache实现的Store接口
type GCacheStore struct {
	cache gcache.Cache
}

// NewGCacheStore 创建GCache存储实例
func NewGCacheStore(size int) *GCacheStore {
	return &GCacheStore{
		cache: gcache.New(size).
			ARC().
			Build(),
	}
}

// NewGCacheStoreWithLRU 创建LRU策略的GCache存储实例
func NewGCacheStoreWithLRU(size int) *GCacheStore {
	return &GCacheStore{
		cache: gcache.New(size).
			LRU().
			Build(),
	}
}

// NewGCacheStoreWithLFU 创建LFU策略的GCache存储实例
func NewGCacheStoreWithLFU(size int) *GCacheStore {
	return &GCacheStore{
		cache: gcache.New(size).
			LFU().
			Build(),
	}
}

// Get 从GCache获取单个值
func (g *GCacheStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	value, err := g.cache.Get(key)
	if err != nil {
		if err == gcache.KeyNotFoundError {
			return false, nil
		}
		return false, err
	}

	data, ok := value.([]byte)
	if !ok {
		return false, nil
	}

	if err := json.Unmarshal(data, dst); err != nil {
		return false, err
	}

	return true, nil
}

// MGet 批量获取值到map中
func (g *GCacheStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	if len(keys) == 0 {
		return nil
	}

	keyType, valueType, err := internal.GetTypeOfMap(dstMap)
	if err != nil {
		return err
	}

	for _, key := range keys {
		value, err := g.cache.Get(key)
		if err != nil {
			if err == gcache.KeyNotFoundError {
				continue
			}
			return err
		}

		data, ok := value.([]byte)
		if !ok {
			continue
		}

		var parsedValue interface{}
		if err := json.Unmarshal(data, &parsedValue); err != nil {
			continue
		}

		if err := internal.SetMapValueWithType(dstMap, key, parsedValue, keyType, valueType); err != nil {
			return err
		}
	}

	return nil
}

// Exists 批量检查键存在性
func (g *GCacheStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	if len(keys) == 0 {
		return make(map[string]bool), nil
	}

	result := make(map[string]bool)
	for _, key := range keys {
		_, err := g.cache.Get(key)
		if err == nil {
			result[key] = true
		} else if err == gcache.KeyNotFoundError {
			result[key] = false
		} else {
			return nil, err
		}
	}

	return result, nil
}

// MSet 批量设置键值对，支持TTL
func (g *GCacheStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	for key, value := range items {
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}

		if ttl > 0 {
			if err := g.cache.SetWithExpire(key, data, ttl); err != nil {
				return err
			}
		} else {
			if err := g.cache.Set(key, data); err != nil {
				return err
			}
		}
	}

	return nil
}

// Del 删除指定键
func (g *GCacheStore) Del(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	var deleted int64
	for _, key := range keys {
		if g.cache.Remove(key) {
			deleted++
		}
	}

	return deleted, nil
}

// Clear 清空缓存
func (g *GCacheStore) Clear() {
	g.cache.Purge()
}

// GCacheStoreOptions GCache存储选项
type GCacheStoreOptions struct {
	Size    int
	Eviction string // "arc", "lru", "lfu"
}

// NewGCacheStoreWithOptions 使用选项创建GCache存储实例
func NewGCacheStoreWithOptions(opts GCacheStoreOptions) *GCacheStore {
	var cache gcache.Cache
	builder := gcache.New(opts.Size)

	switch opts.Eviction {
	case "lru":
		cache = builder.LRU().Build()
	case "lfu":
		cache = builder.LFU().Build()
	default:
		cache = builder.ARC().Build()
	}

	return &GCacheStore{cache: cache}
}

// NewDefaultGCacheStore 创建默认配置的GCache存储实例
func NewDefaultGCacheStore() *GCacheStore {
	return NewGCacheStore(1000)
}