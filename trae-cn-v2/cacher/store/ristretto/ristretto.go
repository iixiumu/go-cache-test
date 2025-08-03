package ristretto

import (
	"context"
	"errors"
	"reflect"
	"time"

	"github.com/dgraph-io/ristretto/v2"
)

// RistrettoStore 实现了Store接口的Ristretto内存缓存后端
type RistrettoStore struct {
	cache *ristretto.Cache[string, interface{}]
}

// NewRistrettoStore 创建一个新的RistrettoStore实例
// maxItems: 最大项目数
func NewRistrettoStore(maxItems int64) (*RistrettoStore, error) {
	cache, err := ristretto.NewCache[string, interface{}](&ristretto.Config[string, interface{}]{
		NumCounters: maxItems * 10,
		MaxCost:     maxItems,
		BufferItems: 64,
	})
	if err != nil {
		return nil, err
	}

	return &RistrettoStore{
		cache: cache,
	}, nil
}

// Get 从Ristretto缓存获取单个值
func (r *RistrettoStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	val, found := r.cache.Get(key)
	if !found {
		return false, nil
	}

	// 使用反射设置值
	dstVal := reflect.ValueOf(dst)
	if dstVal.Kind() != reflect.Ptr {
		return false, errors.New("dst must be a pointer")
	}

	// 确保目标类型与值类型匹配
	dstElem := dstVal.Elem()
	valVal := reflect.ValueOf(val)
	if !valVal.Type().AssignableTo(dstElem.Type()) {
		return false, errors.New("type mismatch")
	}

	dstElem.Set(valVal)
	return true, nil
}

// MGet 批量获取值到map中
func (r *RistrettoStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	// 使用反射确保dstMap是*map[string]T类型
	mapVal := reflect.ValueOf(dstMap)
	if mapVal.Kind() != reflect.Ptr || mapVal.Elem().Kind() != reflect.Map {
		return errors.New("dstMap must be a pointer to a map")
	}

	// 创建结果map
	resultMap := reflect.MakeMap(mapVal.Elem().Type())

	// 填充结果map
	for _, key := range keys {
		val, found := r.cache.Get(key)
		if found {
			// 确保值类型与map值类型匹配
			valVal := reflect.ValueOf(val)
			if valVal.Type().AssignableTo(mapVal.Elem().Type().Elem()) {
				resultMap.SetMapIndex(reflect.ValueOf(key), valVal)
			}
		}
	}

	// 设置结果到dstMap
	mapVal.Elem().Set(resultMap)

	return nil
}

// Exists 批量检查键存在性
func (r *RistrettoStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool, len(keys))

	for _, key := range keys {
		_, result[key] = r.cache.Get(key)
	}

	return result, nil
}

// MSet 批量设置键值对，支持TTL
func (r *RistrettoStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	for key, value := range items {
		// 对于Ristretto，cost始终为1
		// 如果TTL大于0，我们需要使用SetWithTTL
		if ttl > 0 {
			r.cache.SetWithTTL(key, value, 1, ttl)
		} else {
			r.cache.Set(key, value, 1)
		}
	}

	// 等待写入完成
	r.cache.Wait()

	return nil
}

// Del 删除指定键
func (r *RistrettoStore) Del(ctx context.Context, keys ...string) (int64, error) {
	var deleted int64

	for _, key := range keys {
		// 先检查键是否存在
		if _, found := r.cache.Get(key); found {
			r.cache.Del(key)
			deleted++
		}
	}

	return deleted, nil
}