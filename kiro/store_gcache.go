package cache

import (
	"context"
	"reflect"
	"time"

	"github.com/bluele/gcache"
)

// GCacheStore GCache存储实现
type GCacheStore struct {
	cache gcache.Cache
}

// NewGCacheStore 创建GCache存储实例
func NewGCacheStore(cache gcache.Cache) Store {
	return &GCacheStore{
		cache: cache,
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

	return true, copyValue(value, dst)
}

// MGet 批量获取值到map中
func (g *GCacheStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	if len(keys) == 0 {
		return nil
	}

	// 验证dstMap是指向map的指针
	dstValue := reflect.ValueOf(dstMap)
	if dstValue.Kind() != reflect.Ptr || dstValue.Elem().Kind() != reflect.Map {
		return ErrInvalidMapType
	}

	mapValue := dstValue.Elem()

	// 批量获取
	for _, key := range keys {
		if value, err := g.cache.Get(key); err == nil {
			mapValue.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
		}
	}

	return nil
}

// Exists 批量检查键存在性
func (g *GCacheStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool)
	
	for _, key := range keys {
		exists := g.cache.Has(key)
		result[key] = exists
	}

	return result, nil
}

// MSet 批量设置键值对，支持TTL
func (g *GCacheStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	for key, value := range items {
		if ttl > 0 {
			if err := g.cache.SetWithExpire(key, value, ttl); err != nil {
				return err
			}
		} else {
			if err := g.cache.Set(key, value); err != nil {
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
		if g.cache.Has(key) {
			g.cache.Remove(key)
			deleted++
		}
	}

	return deleted, nil
}