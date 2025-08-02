package cache

import (
	"context"
	"time"

	"github.com/bluele/gcache"
)

// gcacheStore gcache存储实现
type gcacheStore struct {
	cache gcache.Cache
}

// NewGCacheStore 创建gcache存储实例
func NewGCacheStore(cache gcache.Cache) Store {
	return &gcacheStore{
		cache: cache,
	}
}

// Get 从gcache获取单个值
func (g *gcacheStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	value, err := g.cache.Get(key)
	if err != nil {
		if err == gcache.KeyNotFoundError {
			return false, nil
		}
		return false, err
	}
	
	// 确保值是字节切片
	data, ok := value.([]byte)
	if !ok {
		return false, nil
	}
	
	return true, deserializeValue(data, dst)
}

// MGet 批量获取值
func (g *gcacheStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	if len(keys) == 0 {
		return nil
	}
	
	data := make(map[string][]byte)
	for _, key := range keys {
		if value, err := g.cache.Get(key); err == nil {
			if dataBytes, ok := value.([]byte); ok {
				data[key] = dataBytes
			}
		}
	}
	
	return deserializeMap(data, dstMap)
}

// Exists 批量检查键存在性
func (g *gcacheStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool, len(keys))
	
	for _, key := range keys {
		_, err := g.cache.Get(key)
		result[key] = err == nil
	}
	
	return result, nil
}

// MSet 批量设置键值对
func (g *gcacheStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}
	
	// 序列化所有值
	serialized, err := serializeMap(items)
	if err != nil {
		return err
	}
	
	// gcache不支持TTL，忽略ttl参数
	for key, value := range serialized {
		g.cache.Set(key, value)
	}
	
	return nil
}

// Del 删除指定键
func (g *gcacheStore) Del(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}
	
	var deleted int64
	for _, key := range keys {
		g.cache.Remove(key)
		deleted++
	}
	
	return deleted, nil
}