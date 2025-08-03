package ristretto

import (
	"context"
	"errors"
	"reflect"
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"go-cache/cacher/store"
)

// ristrettoStore 是 Store 接口的 Ristretto 实现
type ristrettoStore struct {
	cache *ristretto.Cache[string, interface{}]
}

// NewRistrettoStore 创建一个新的 Ristretto 存储实例
func NewRistrettoStore(numCounters int64) (store.Store, error) {
	cache, err := ristretto.NewCache(&ristretto.Config[string, interface{}]{
		NumCounters: numCounters,
		MaxCost:     10000, // 默认最大成本
		BufferItems: 64,    // 默认缓冲项目数
	})
	if err != nil {
		return nil, err
	}

	return &ristrettoStore{
		cache: cache,
	}, nil
}

// Get 从 Ristretto 获取单个值
func (r *ristrettoStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	value, found := r.cache.Get(key)
	if !found {
		return false, nil
	}

	// 使用反射将值复制到目标变量
	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() != reflect.Ptr {
		return false, errors.New("dst must be a pointer")
	}

	// 获取目标变量的类型
	dstElem := dstValue.Elem()
	valueReflect := reflect.ValueOf(value)

	// 确保类型兼容
	if valueReflect.Type().AssignableTo(dstElem.Type()) {
		dstElem.Set(valueReflect)
	} else if valueReflect.Type().ConvertibleTo(dstElem.Type()) {
		dstElem.Set(valueReflect.Convert(dstElem.Type()))
	} else {
		return false, errors.New("value type is not compatible with dst type")
	}

	return true, nil
}

// MGet 批量从 Ristretto 获取值
func (r *ristrettoStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	// 使用反射处理目标 map
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr || dstMapValue.Elem().Kind() != reflect.Map {
		return errors.New("dstMap must be a pointer to a map")
	}

	mapValue := dstMapValue.Elem()
	mapValue.Set(reflect.MakeMap(mapValue.Type()))

	keyType := mapValue.Type().Key()
	elemType := mapValue.Type().Elem()

	for _, key := range keys {
		if value, found := r.cache.Get(key); found {
			// 创建 map 元素
			elemValue := reflect.ValueOf(value)

			// 确保类型兼容
			if !elemValue.Type().AssignableTo(elemType) {
				// 尝试转换类型
				if elemValue.Type().ConvertibleTo(elemType) {
					elemValue = elemValue.Convert(elemType)
				} else {
					continue // 跳过不兼容的类型
				}
			}

			// 设置 map 值
			mapValue.SetMapIndex(reflect.ValueOf(key).Convert(keyType), elemValue)
		}
	}

	return nil
}

// Exists 检查键在 Ristretto 中是否存在
func (r *ristrettoStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	results := make(map[string]bool)
	for _, key := range keys {
		_, found := r.cache.Get(key)
		results[key] = found
	}
	return results, nil
}

// MSet 批量设置键值对到 Ristretto
func (r *ristrettoStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	// 注意：Ristretto 不直接支持 TTL，但我们可以通过记录设置时间并在 Get 时检查来模拟
	// 在这个简化实现中，我们忽略 TTL 参数
	for key, value := range items {
		// 在 Ristretto 中，成本固定为 1
		r.cache.Set(key, value, 1)
	}
	// 等待写入完成
	r.cache.Wait()
	return nil
}

// Del 从 Ristretto 删除指定键
func (r *ristrettoStore) Del(ctx context.Context, keys ...string) (int64, error) {
	var count int64
	for _, key := range keys {
		r.cache.Del(key)
		count++
	}
	return count, nil
}