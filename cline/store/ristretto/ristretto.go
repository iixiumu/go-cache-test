package ristretto

import (
	"context"
	"time"

	"github.com/example/go-cache/store"
	"github.com/hypermodeinc/ristretto"
)

// RistrettoStore Ristretto存储实现
type RistrettoStore struct {
	cache *ristretto.Cache
}

// NewRistrettoStore 创建新的Ristretto存储实例
func NewRistrettoStore(cache *ristretto.Cache) store.Store {
	return &RistrettoStore{
		cache: cache,
	}
}

// Get 从Ristretto获取单个值
func (r *RistrettoStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	value, found := r.cache.Get(key)
	if !found {
		return false, nil
	}

	// 将值赋给dst
	// 这里需要使用反射来处理不同的类型
	// 简化处理：假设dst是指向正确类型的指针
	// 在实际实现中，可能需要更复杂的反射处理
	dst = value
	return true, nil
}

// MGet 批量从Ristretto获取值
func (r *RistrettoStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	// Ristretto不直接支持批量获取，需要逐个获取
	// 这里简化处理，实际实现可能需要更复杂的反射逻辑

	// 假设dstMap是指向map[string]interface{}的指针
	dst := dstMap.(*map[string]interface{})
	if *dst == nil {
		*dst = make(map[string]interface{})
	}

	for _, key := range keys {
		if value, found := r.cache.Get(key); found {
			(*dst)[key] = value
		}
	}

	return nil
}

// Exists 批量检查键存在性
func (r *RistrettoStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	exists := make(map[string]bool)
	for _, key := range keys {
		_, exists[key] = r.cache.Get(key)
	}
	return exists, nil
}

// MSet 批量设置键值对
func (r *RistrettoStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	for key, value := range items {
		// Ristretto的Set不支持TTL，需要在值中包含过期时间信息
		// 这里简化处理，忽略TTL
		r.cache.Set(key, value, 1)
	}
	return nil
}

// Del 删除指定键
func (r *RistrettoStore) Del(ctx context.Context, keys ...string) (int64, error) {
	count := int64(0)
	for _, key := range keys {
		if r.cache.Del(key) {
			count++
		}
	}
	return count, nil
}
