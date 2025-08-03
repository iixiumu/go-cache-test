package ristretto

import (
	"context"
	"fmt"
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"go-cache/cacher/store"
)

// RistrettoStore Ristretto存储实现
type RistrettoStore struct {
	cache *ristretto.Cache[string, interface{}]
}

// NewRistrettoStore 创建新的Ristretto存储实例
func NewRistrettoStore(cache *ristretto.Cache[string, interface{}]) *RistrettoStore {
	return &RistrettoStore{cache: cache}
}

// Get 从Ristretto获取单个值
func (r *RistrettoStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	value, found := r.cache.Get(key)
	if !found {
		return false, nil
	}

	// 将值复制到dst
	// 由于Ristretto直接存储对象，不需要反序列化
	// 这里简单地将值赋给dst
	// 在实际应用中可能需要更复杂的类型转换
	switch v := dst.(type) {
	case *interface{}:
		*v = value
	case *string:
		if str, ok := value.(string); ok {
			*v = str
		} else {
			return false, nil
		}
	case *int:
		if i, ok := value.(int); ok {
			*v = i
		} else {
			return false, nil
		}
	case *float64:
		if f, ok := value.(float64); ok {
			*v = f
		} else {
			return false, nil
		}
	default:
		// 对于其他类型，直接赋值
		// 在实际应用中可能需要更复杂的处理
		*(&dst) = value
	}

	return true, nil
}

// MGet 批量获取值到map中
func (r *RistrettoStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	resultMap := make(map[string]interface{})
	for _, key := range keys {
		if value, found := r.cache.Get(key); found {
			resultMap[key] = value
		}
	}

	// 将结果复制到dstMap
	// 使用类型断言来处理不同的map类型
	switch m := dstMap.(type) {
	case *map[string]interface{}:
		*m = resultMap
	case *map[string]string:
		stringMap := make(map[string]string)
		for k, v := range resultMap {
			if str, ok := v.(string); ok {
				stringMap[k] = str
			} else {
				// 如果值不是字符串，将其转换为字符串
				stringMap[k] = fmt.Sprintf("%v", v)
			}
		}
		*m = stringMap
	default:
		// 对于其他类型，尝试直接赋值
		*(&dstMap) = resultMap
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
	cost := int64(1)
	for key, value := range items {
		// Ristretto的SetWithTTL方法需要一个cost参数
		// 这里简单地设置为1
		if ttl > 0 {
			r.cache.SetWithTTL(key, value, cost, ttl)
		} else {
			r.cache.Set(key, value, cost)
		}
	}

	// 等待写入完成
	r.cache.Wait()

	return nil
}

// Del 删除指定键
func (r *RistrettoStore) Del(ctx context.Context, keys ...string) (int64, error) {
	for _, key := range keys {
		r.cache.Del(key)
	}

	return int64(len(keys)), nil
}

// 确保RistrettoStore实现了Store接口
var _ store.Store = (*RistrettoStore)(nil)