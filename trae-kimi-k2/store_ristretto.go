package cache

import (
	"context"
	"time"

	"github.com/dgraph-io/ristretto"
)

// ristrettoStore Ristretto存储实现
type ristrettoStore struct {
	cache *ristretto.Cache
}

// NewRistrettoStore 创建Ristretto存储实例
func NewRistrettoStore(cache *ristretto.Cache) Store {
	return &ristrettoStore{
		cache: cache,
	}
}

// Get 从Ristretto获取单个值
func (r *ristrettoStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	value, found := r.cache.Get(key)
	if !found {
		return false, nil
	}
	
	// 确保值是字节切片
	data, ok := value.([]byte)
	if !ok {
		return false, nil
	}
	
	return true, deserializeValue(data, dst)
}

// MGet 批量获取值
func (r *ristrettoStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	if len(keys) == 0 {
		return nil
	}
	
	data := make(map[string][]byte)
	for _, key := range keys {
		if value, found := r.cache.Get(key); found {
			if dataBytes, ok := value.([]byte); ok {
				data[key] = dataBytes
			}
		}
	}
	
	return deserializeMap(data, dstMap)
}

// Exists 批量检查键存在性
func (r *ristrettoStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool, len(keys))
	
	for _, key := range keys {
		_, found := r.cache.Get(key)
		result[key] = found
	}
	
	return result, nil
}

// MSet 批量设置键值对
func (r *ristrettoStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}
	
	// 序列化所有值
	serialized, err := serializeMap(items)
	if err != nil {
		return err
	}
	
	// 计算TTL（转换为纳秒）
	var ttlNs int64
	if ttl > 0 {
		ttlNs = int64(ttl)
	} else {
		ttlNs = 0
	}
	
	// 批量设置
	for key, value := range serialized {
		r.cache.SetWithTTL(key, value, 1, time.Duration(ttlNs))
	}
	
	return nil
}

// Del 删除指定键
func (r *ristrettoStore) Del(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}
	
	var deleted int64
	for _, key := range keys {
		r.cache.Del(key)
		deleted++
	}
	
	return deleted, nil
}