package cache

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/dgraph-io/ristretto"
)

// ristrettoStore 实现了Store接口，使用Ristretto作为存储后端
type ristrettoStore struct {
	cache *ristretto.Cache
}

// NewRistrettoStore 创建一个新的Ristretto Store实例
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

	// 将值序列化为字节
	valBytes, ok := value.([]byte)
	if !ok {
		return false, nil
	}

	// 反序列化值到dst
	err := json.Unmarshal(valBytes, dst)
	if err != nil {
		return false, err
	}

	return true, nil
}

// MGet 批量从Ristretto获取值
func (r *ristrettoStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	// 确保dstMap是指向map的指针
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr || dstMapValue.Elem().Kind() != reflect.Map {
		return nil
	}

	// 获取map的元素类型
	dstMapElem := dstMapValue.Elem()
	mapValueType := dstMapElem.Type().Elem()

	// 批量获取值
	for _, key := range keys {
		value, found := r.cache.Get(key)
		if !found {
			continue
		}

		// 将值序列化为字节
		valBytes, ok := value.([]byte)
		if !ok {
			continue
		}

		// 创建对应类型的值
		valuePtr := reflect.New(mapValueType).Interface()

		// 反序列化
		err := json.Unmarshal(valBytes, valuePtr)
		if err != nil {
			continue
		}

		// 设置map值
		dstMapElem.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(valuePtr).Elem())
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
		valBytes, err := json.Marshal(value)
		if err != nil {
			return err
		}

		// 设置值到缓存，Ristretto没有TTL设置，所以ttl参数被忽略
		// 等待缓存操作完成
		r.cache.Set(key, valBytes, 1)
		r.cache.Wait()
	}

	return nil
}

// Del 从Ristretto删除指定键
func (r *ristrettoStore) Del(ctx context.Context, keys ...string) (int64, error) {
	var deleted int64
	for _, key := range keys {
		r.cache.Del(key)
		deleted++
	}
	return deleted, nil
}
