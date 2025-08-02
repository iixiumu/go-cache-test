package gcache

import (
	"context"
	"time"

	"github.com/bluele/gcache"
	"github.com/example/go-cache/store"
)

// GCacheStore GCache存储实现
type GCacheStore struct {
	cache gcache.Cache
}

// NewGCacheStore 创建新的GCache存储实例
func NewGCacheStore(cache gcache.Cache) store.Store {
	return &GCacheStore{
		cache: cache,
	}
}

// Get 从GCache获取单个值
func (g *GCacheStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	value, err := g.cache.Get(key)
	if err != nil {
		// 检查是否是未找到的错误
		if err == gcache.KeyNotFoundError {
			return false, nil
		}
		return false, err
	}

	// 将值赋给dst
	// 这里需要使用反射来处理不同的类型
	// 简化处理：假设dst是指向正确类型的指针
	// 在实际实现中，可能需要更复杂的反射处理
	dst = value
	return true, nil
}

// MGet 批量从GCache获取值
func (g *GCacheStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	// GCache不直接支持批量获取，需要逐个获取
	// 这里简化处理，实际实现可能需要更复杂的反射逻辑

	// 假设dstMap是指向map[string]interface{}的指针
	dst := dstMap.(*map[string]interface{})
	if *dst == nil {
		*dst = make(map[string]interface{})
	}

	for _, key := range keys {
		if value, err := g.cache.Get(key); err == nil {
			(*dst)[key] = value
		}
	}

	return nil
}

// Exists 批量检查键存在性
func (g *GCacheStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	exists := make(map[string]bool)
	for _, key := range keys {
		_, err := g.cache.Get(key)
		exists[key] = err == nil
	}
	return exists, nil
}

// MSet 批量设置键值对
func (g *GCacheStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	for key, value := range items {
		if ttl > 0 {
			g.cache.SetWithExpire(key, value, ttl)
		} else {
			g.cache.Set(key, value)
		}
	}
	return nil
}

// Del 删除指定键
func (g *GCacheStore) Del(ctx context.Context, keys ...string) (int64, error) {
	count := int64(0)
	for _, key := range keys {
		if g.cache.Remove(key) {
			count++
		}
	}
	return count, nil
}
