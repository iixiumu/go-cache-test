package store

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/dgraph-io/ristretto"
)

// ristrettoStore 是基于Ristretto的Store实现
type ristrettoStore struct {
	cache *ristretto.Cache
}

// NewRistrettoStore 创建一个新的Ristretto存储实例
func NewRistrettoStore(cache *ristretto.Cache) Store {
	return &ristrettoStore{
		cache: cache,
	}
}

// Get 从Ristretto获取单个值
func (r *ristrettoStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	val, found := r.cache.Get(key)
	if !found {
		return false, nil
	}

	// val是字节切片，需要反序列化
	data, ok := val.([]byte)
	if !ok {
		return false, nil
	}

	// 反序列化到dst
	err := json.Unmarshal(data, dst)
	if err != nil {
		return false, err
	}

	return true, nil
}

// MGet 批量从Ristretto获取值
func (r *ristrettoStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	// 使用反射将结果设置到dstMap
	// dstMap应该是指向map的指针
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr || dstMapValue.IsNil() {
		return nil
	}

	dstMapElem := dstMapValue.Elem()
	if dstMapElem.Kind() != reflect.Map {
		return nil
	}

	// 获取map的键和值类型
	mapKeyType := dstMapElem.Type().Key()
	mapValueType := dstMapElem.Type().Elem()

	// 批量获取值
	for _, key := range keys {
		val, found := r.cache.Get(key)
		if !found {
			continue
		}

		// val是字节切片，需要反序列化
		data, ok := val.([]byte)
		if !ok {
			continue
		}

		// 创建map键
		mapKey := reflect.ValueOf(key).Convert(mapKeyType)

		// 创建map值
		mapValue := reflect.New(mapValueType).Interface()
		err := json.Unmarshal(data, mapValue)
		if err != nil {
			continue
		}

		// 设置map元素
		dstMapElem.SetMapIndex(mapKey, reflect.ValueOf(mapValue).Elem())
	}

	return nil
}

// Exists 批量检查键在Ristretto中的存在性
func (r *ristrettoStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool)

	for _, key := range keys {
		_, found := r.cache.Get(key)
		result[key] = found
	}

	return result, nil
}

// MSet 批量设置键值对到Ristretto
func (r *ristrettoStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	for key, value := range items {
		// 序列化值
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}

		// 计算成本（字节长度）
		cost := int64(len(data))

		// 设置到缓存中
		r.cache.SetWithTTL(key, data, cost, ttl)
	}

	return nil
}

// Del 从Ristretto删除指定键
func (r *ristrettoStore) Del(ctx context.Context, keys ...string) (int64, error) {
	count := int64(0)

	for _, key := range keys {
		// Ristretto没有直接返回删除数量的API，我们假设每个键都存在
		r.cache.Del(key)
		count++
	}

	return count, nil
}
