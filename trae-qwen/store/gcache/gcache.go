package gcache

import (
	"context"
	"time"
	"github.com/bluele/gcache"
	"github.com/xiumu/go-cache/store"
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

// Get 从GCache存储中获取单个值
func (g *GCacheStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	// 在实际实现中，需要从GCache中获取值并处理序列化/反序列化
	// 这里简化实现
	return false, nil
}

// MGet 批量获取值到map中
func (g *GCacheStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	// 在实际实现中，需要从GCache中批量获取值并处理序列化/反序列化
	return nil
}

// Exists 批量检查键存在性
func (g *GCacheStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool)
	for _, key := range keys {
		// GCache没有直接的Exists方法，需要通过Get来判断
		_, err := g.cache.Get(key)
		result[key] = err == nil
	}
	return result, nil
}

// MSet 批量设置键值对，支持TTL
func (g *GCacheStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	// 在实际实现中，需要将值设置到GCache中并处理序列化
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
	var deleted int64
	for _, key := range keys {
		if g.cache.Remove(key) {
			deleted++
		}
	}
	return deleted, nil
}