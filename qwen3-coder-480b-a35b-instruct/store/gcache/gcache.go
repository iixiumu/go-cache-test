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

// New 创建一个新的GCache存储实例
func New(cache gcache.Cache) store.Store {
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
	// 这里需要进行类型转换

	return true, nil
}

// MGet 批量从GCache获取值
func (g *GCacheStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	// 批量获取值
	// 这里需要实现具体的批量获取逻辑

	return nil
}

// Exists 批量检查键存在性
func (g *GCacheStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool)
	
	// 检查每个键是否存在
	for _, key := range keys {
		_, err := g.cache.Get(key)
		result[key] = err == nil
	}

	return result, nil
}

// MSet 批量设置键值对
func (g *GCacheStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	// 批量设置键值对
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
	
	// 删除指定的键
	for _, key := range keys {
		if g.cache.Remove(key) {
			count++
		}
	}

	return count, nil
}