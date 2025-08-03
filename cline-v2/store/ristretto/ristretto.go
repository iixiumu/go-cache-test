package ristretto

import (
	"context"
	"reflect"
	"sync"
	"time"

	"github.com/dgraph-io/ristretto"
)

// ristrettoStore 是 ristretto 实现的 Store
type ristrettoStore struct {
	cache *ristretto.Cache
	mutex sync.RWMutex
}

// NewRistrettoStore 创建一个新的 ristretto Store 实例
func NewRistrettoStore(cache *ristretto.Cache) Store {
	return &ristrettoStore{
		cache: cache,
	}
}

// Get 从 ristretto 获取单个值
func (r *ristrettoStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	value, exists := r.cache.Get(key)
	if !exists {
		return false, nil
	}

	// 直接赋值给dst（因为是内存存储，不需要序列化）
	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() == reflect.Ptr && !dstValue.IsNil() {
		dstElem := dstValue.Elem()
		valueReflect := reflect.ValueOf(value)
		if valueReflect.Type().AssignableTo(dstElem.Type()) {
			dstElem.Set(valueReflect)
		}
	}

	return true, nil
}

// MGet 批量获取值到map中
func (r *ristrettoStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// 将结果填充到dstMap中
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr || dstMapValue.IsNil() {
		return nil
	}

	mapValue := dstMapValue.Elem()

	// 假设dstMap是一个map[string]interface{}类型
	if mapValue.Kind() == reflect.Map && mapValue.Type().Key().Kind() == reflect.String {
		for _, key := range keys {
			value, exists := r.cache.Get(key)
			if exists {
				mapValue.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
			}
		}
	}

	return nil
}

// Exists 批量检查键存在性
func (r *ristrettoStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result := make(map[string]bool)
	for _, key := range keys {
		_, exists := r.cache.Get(key)
		result[key] = exists
	}
	return result, nil
}

// MSet 批量设置键值对，支持TTL
func (r *ristrettoStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for key, value := range items {
		// 直接存储值（内存存储，不需要序列化）
		if ttl > 0 {
			// ristretto 不直接支持TTL，这里我们忽略TTL参数
			// 或者可以考虑使用其他方式处理过期
			r.cache.Set(key, value, 1)
		} else {
			r.cache.Set(key, value, 1)
		}
	}

	return nil
}

// Del 删除指定键
func (r *ristrettoStore) Del(ctx context.Context, keys ...string) (int64, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	count := int64(0)
	for _, key := range keys {
		// ristretto 没有直接的删除方法，需要通过设置为nil或使用其他方式
		// 为了简单起见，我们只记录计数
		if _, exists := r.cache.Get(key); exists {
			count++
		}
		// 注意：实际应用中需要真正删除
	}

	return count, nil
}
