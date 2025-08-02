package store

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/xiumu/go-cache/cache"
)

// ristrettoStore Ristretto存储实现
type ristrettoStore struct {
	cache *ristretto.Cache
}

// NewRistrettoStore 创建一个新的Ristretto存储实例
func NewRistrettoStore(cache *ristretto.Cache) cache.Store {
	return &ristrettoStore{
		cache: cache,
	}
}

// Get 从存储后端获取单个值
func (r *ristrettoStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	// 从Ristretto缓存中获取值
	val, found := r.cache.Get(key)
	if !found {
		return false, nil
	}

	// 类型断言为字节切片
	data, ok := val.([]byte)
	if !ok {
		return false, nil
	}

	// 反序列化数据到dst
	return true, json.Unmarshal(data, dst)
}

// MGet 批量获取值到map中
func (r *ristrettoStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	// 检查dstMap是否为指针
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr {
		return nil
	}

	// 获取map的实际值
	dstMapValue = dstMapValue.Elem()
	if dstMapValue.Kind() != reflect.Map {
		return nil
	}

	// 获取map的元素类型
	mapElemType := dstMapValue.Type().Elem()

	// 遍历所有键
	for _, key := range keys {
		// 从Ristretto缓存中获取值
		val, found := r.cache.Get(key)
		if !found {
			continue
		}

		// 类型断言为字节切片
		data, ok := val.([]byte)
		if !ok {
			continue
		}

		// 创建一个新的元素实例
		elem := reflect.New(mapElemType).Interface()

		// 反序列化数据
		err := json.Unmarshal(data, elem)
		if err != nil {
			continue
		}

		// 将值设置到map中
		dstMapValue.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(elem).Elem())
	}

	return nil
}

// Exists 批量检查键存在性
func (r *ristrettoStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool)

	// 遍历所有键
	for _, key := range keys {
		// 检查键是否存在
		_, found := r.cache.Get(key)
		result[key] = found
	}

	return result, nil
}

// MSet 批量设置键值对，支持TTL
func (r *ristrettoStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	// 遍历所有键值对
	for key, value := range items {
		// 序列化值
		data, err := json.Marshal(value)
		if err != nil {
			continue
		}

		// 设置键值对到Ristretto缓存中
		// Ristretto的SetWithTTL方法需要一个cost参数，我们这里使用1作为默认值
		r.cache.SetWithTTL(key, data, 1, ttl)
	}

	return nil
}

// Del 删除指定键
func (r *ristrettoStore) Del(ctx context.Context, keys ...string) (int64, error) {
	var deleted int64

	// 遍历所有键
	for _, key := range keys {
		// 从缓存中删除键
		r.cache.Del(key)
		// Ristretto的Del方法没有返回值，我们假设删除成功
		deleted++
	}

	return deleted, nil
}