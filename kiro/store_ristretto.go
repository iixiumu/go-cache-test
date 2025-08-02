package cache

import (
	"context"
	"reflect"
	"sync"
	"time"

	"github.com/dgraph-io/ristretto"
)

// RistrettoStore Ristretto存储实现
type RistrettoStore struct {
	cache *ristretto.Cache
	mu    sync.RWMutex
}

// NewRistrettoStore 创建Ristretto存储实例
func NewRistrettoStore(cache *ristretto.Cache) Store {
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

	return true, copyValue(value, dst)
}

// MGet 批量获取值到map中
func (r *RistrettoStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
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
		if value, found := r.cache.Get(key); found {
			mapValue.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
		}
	}

	return nil
}

// Exists 批量检查键存在性
func (r *RistrettoStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool)
	
	for _, key := range keys {
		_, found := r.cache.Get(key)
		result[key] = found
	}

	return result, nil
}

// MSet 批量设置键值对，支持TTL
func (r *RistrettoStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	for key, value := range items {
		cost := int64(1) // 简单的成本计算，实际使用中可能需要更复杂的逻辑
		
		if ttl > 0 {
			r.cache.SetWithTTL(key, value, cost, ttl)
		} else {
			r.cache.Set(key, value, cost)
		}
	}

	// 等待所有设置操作完成
	r.cache.Wait()
	return nil
}

// Del 删除指定键
func (r *RistrettoStore) Del(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	var deleted int64
	for _, key := range keys {
		if _, found := r.cache.Get(key); found {
			r.cache.Del(key)
			deleted++
		}
	}

	return deleted, nil
}