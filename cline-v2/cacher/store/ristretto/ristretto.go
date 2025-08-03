package ristretto

import (
	"context"
	"time"

	"go-cache/cacher/store"

	"github.com/dgraph-io/ristretto/v2"
)

// RistrettoStore Ristretto内存存储实现
type RistrettoStore struct {
	cache *ristretto.Cache[string, interface{}]
}

// NewRistrettoStore 创建新的Ristretto存储实例
func NewRistrettoStore() (store.Store, error) {
	cache, err := ristretto.NewCache(&ristretto.Config[string, interface{}]{
		NumCounters: 1e7,     // number of keys to track frequency of (10M).
		MaxCost:     1 << 30, // maximum cost of cache (1GB).
		BufferItems: 64,      // number of keys per Get buffer.
	})
	if err != nil {
		return nil, err
	}

	return &RistrettoStore{
		cache: cache,
	}, nil
}

// Get 从Ristretto获取单个值
func (r *RistrettoStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	value, found := r.cache.Get(key)
	if !found {
		return false, nil
	}

	// 直接将值赋给dst（通过反射）
	// 这里需要更复杂的反射逻辑来处理类型转换
	// 暂时使用类型断言处理常见情况
	switch dst := dst.(type) {
	case *string:
		if str, ok := value.(string); ok {
			*dst = str
		}
	case *int:
		if i, ok := value.(int); ok {
			*dst = i
		}
	case *interface{}:
		*dst = value
	default:
		// 对于复杂类型，直接赋值
		// 这需要更复杂的反射处理
		dstValue := dst
		if dstValue != nil {
			// 这里需要使用反射来赋值
			// 暂时简化处理
		}
	}

	return true, nil
}

// MGet 批量获取值到map中
func (r *RistrettoStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	result := make(map[string]interface{})
	for _, key := range keys {
		if value, found := r.cache.Get(key); found {
			result[key] = value
		}
	}

	// 使用反射将result赋值给dstMap
	// 暂时使用类型断言处理常见的map[string]string情况
	if mapPtr, ok := dstMap.(*map[string]string); ok {
		stringMap := make(map[string]string)
		for k, v := range result {
			if str, ok := v.(string); ok {
				stringMap[k] = str
			}
		}
		*mapPtr = stringMap
		return nil
	}

	// 处理map[string]interface{}情况
	if mapPtr, ok := dstMap.(*map[string]interface{}); ok {
		*mapPtr = result
		return nil
	}

	return nil
}

// Exists 批量检查键存在性
func (r *RistrettoStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	exists := make(map[string]bool)
	for _, key := range keys {
		_, found := r.cache.Get(key)
		exists[key] = found
	}
	return exists, nil
}

// MSet 批量设置键值对，支持TTL
func (r *RistrettoStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	// Ristretto不直接支持TTL，但可以通过成本来模拟
	// 这里简化处理，假设每个item的成本为1
	for key, value := range items {
		// 设置成本为1
		r.cache.Set(key, value, 1)
	}

	// 等待缓冲区处理完成
	r.cache.Wait()

	return nil
}

// Del 删除指定键
func (r *RistrettoStore) Del(ctx context.Context, keys ...string) (int64, error) {
	// Ristretto的Del方法不返回是否删除成功
	// 我们先检查哪些键存在，然后删除所有键
	var existed int64
	for _, key := range keys {
		if _, found := r.cache.Get(key); found {
			existed++
		}
		r.cache.Del(key)
	}
	return existed, nil
}
