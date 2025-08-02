package gcache

import (
	"context"
	"fmt"
	"time"

	"github.com/bluele/gcache"
	cache "go-cache"
)

// GCacheStore GCache存储后端实现
type GCacheStore struct {
	cache gcache.Cache
}

// NewGCacheStore 创建GCache存储实例
func NewGCacheStore(size int, gcType string) *GCacheStore {
	var c gcache.Cache
	
	switch gcType {
	case "lru":
		c = gcache.New(size).LRU().Build()
	case "lfu":
		c = gcache.New(size).LFU().Build()
	case "arc":
		c = gcache.New(size).ARC().Build()
	default:
		c = gcache.New(size).LRU().Build()
	}

	return &GCacheStore{
		cache: c,
	}
}

// NewLRUGCacheStore 创建LRU类型的GCache存储实例
func NewLRUGCacheStore(size int) *GCacheStore {
	return NewGCacheStore(size, "lru")
}

// Get 从GCache获取单个值
func (g *GCacheStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	value, err := g.cache.Get(key)
	if err != nil {
		if err == gcache.KeyNotFoundError {
			return false, nil
		}
		return false, fmt.Errorf("gcache get failed: %w", err)
	}

	if strValue, ok := value.(string); ok {
		if err := cache.DeserializeValue(strValue, dst); err != nil {
			return false, fmt.Errorf("deserialize failed: %w", err)
		}
		return true, nil
	}

	return false, fmt.Errorf("invalid value type in cache: %T", value)
}

// MGet 批量获取值到map中
func (g *GCacheStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	if len(keys) == 0 {
		return nil
	}

	if err := cache.ValidateMapPointer(dstMap); err != nil {
		return err
	}

	for _, key := range keys {
		value, err := g.cache.Get(key)
		if err == nil {
			if strValue, ok := value.(string); ok {
				if err := cache.SetMapValue(dstMap, key, strValue); err != nil {
					return fmt.Errorf("set map value failed for key %s: %w", key, err)
				}
			}
		} else if err != gcache.KeyNotFoundError {
			return fmt.Errorf("gcache get failed for key %s: %w", key, err)
		}
	}

	return nil
}

// Exists 批量检查键存在性
func (g *GCacheStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool, len(keys))
	
	for _, key := range keys {
		_, err := g.cache.Get(key)
		if err == nil {
			result[key] = true
		} else if err == gcache.KeyNotFoundError {
			result[key] = false
		} else {
			return nil, fmt.Errorf("gcache exists check failed for key %s: %w", key, err)
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
		serialized, err := cache.SerializeValue(value)
		if err != nil {
			return fmt.Errorf("serialize failed for key %s: %w", key, err)
		}

		if ttl > 0 {
			err = g.cache.SetWithExpire(key, serialized, ttl)
		} else {
			err = g.cache.Set(key, serialized)
		}
		
		if err != nil {
			return fmt.Errorf("gcache set failed for key %s: %w", key, err)
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
		// 检查键是否存在
		if _, err := g.cache.Get(key); err == nil {
			if g.cache.Remove(key) {
				deleted++
			}
		}
	}

	return deleted, nil
}

// Len 返回缓存中的项目数量
func (g *GCacheStore) Len() int {
	return g.cache.Len(false)
}

// Purge 清空所有缓存项
func (g *GCacheStore) Purge() {
	g.cache.Purge()
}