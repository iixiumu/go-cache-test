package ristretto

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"github.com/dgraph-io/ristretto" 
	"github.com/xiumu/git/me/go-cache/trae/pkg/store"
)

// RistrettoStore 基于Ristretto的Store实现
type RistrettoStore struct {
	cache *ristretto.Cache
}

// NewRistrettoStore 创建一个新的RistrettoStore实例
func NewRistrettoStore(cache *ristretto.Cache) store.Store {
	return &RistrettoStore{
		cache: cache,
	}
}

// Get 从Ristretto缓存获取单个值
func (r *RistrettoStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	val, found := r.cache.Get(key)
	if !found {
		return false, nil
	}

	// 将值反序列化到dst
	valBytes, err := json.Marshal(val)
	if err != nil {
		return false, err
	}

	return true, json.Unmarshal(valBytes, dst)
}

// MGet 批量从Ristretto缓存获取值
func (r *RistrettoStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	// 检查dstMap是否为*map[string]T类型
	mapValue := reflect.ValueOf(dstMap)
	if mapValue.Kind() != reflect.Ptr || mapValue.Elem().Kind() != reflect.Map {
		return errors.New("dstMap must be a pointer to a map")
	}

	// 创建结果map
	resultMap := make(map[string]interface{})
	for _, key := range keys {
		if val, found := r.cache.Get(key); found {
			resultMap[key] = val
		}
	}

	// 将结果设置到dstMap
	return setMapValue(dstMap, resultMap)
}

// Exists 批量检查键存在性
func (r *RistrettoStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool)
	for _, key := range keys {
		_, result[key] = r.cache.Get(key)
	}
	return result, nil
}

// MSet 批量设置键值对，支持TTL
func (r *RistrettoStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	for key, value := range items {
		// 存储值
		if ttl > 0 {
			r.cache.SetWithTTL(key, value, 1, ttl)
		} else {
			r.cache.Set(key, value, 1)
		}
	}
	return nil
}

// Del 删除指定键
func (r *RistrettoStore) Del(ctx context.Context, keys ...string) (int64, error) {
	var count int64
	for _, key := range keys {
		r.cache.Del(key)
		count++
	}
	return count, nil
}

// 辅助函数：使用反射设置map值
func setMapValue(dstMap interface{}, srcMap map[string]interface{}) error {
	// 获取dstMap的反射值
	dstValue := reflect.ValueOf(dstMap).Elem()

	// 清空dstMap
	dstValue.Set(reflect.MakeMap(dstValue.Type()))

	// 获取map的键类型和值类型
	keyType := dstValue.Type().Key()
	valueType := dstValue.Type().Elem()

	// 遍历srcMap，设置到dstMap
	for k, v := range srcMap {
		// 转换键类型
		keyValue := reflect.ValueOf(k)
		if !keyValue.Type().AssignableTo(keyType) {
			return errors.New("key type mismatch")
		}

		// 转换值类型
		valueValue := reflect.ValueOf(v)
		if !valueValue.Type().AssignableTo(valueType) {
			// 尝试转换类型
			if !valueValue.Type().ConvertibleTo(valueType) {
				return errors.New("value type mismatch and cannot be converted")
			}
			valueValue = valueValue.Convert(valueType)
		}

		// 设置到map
		dstValue.SetMapIndex(keyValue, valueValue)
	}

	return nil
}