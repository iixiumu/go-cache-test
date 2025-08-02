package gcache

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"github.com/bluele/gcache"
	"github.com/xiumu/git/me/go-cache/trae/pkg/store"
)

// GCacheStore 基于gcache的Store实现
type GCacheStore struct {
	cache gcache.Cache
}

// NewGCacheStore 创建一个新的GCacheStore实例
func NewGCacheStore(cache gcache.Cache) store.Store {
	return &GCacheStore{
		cache: cache,
	}
}

// Get 从gcache获取单个值
func (g *GCacheStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	val, err := g.cache.Get(key)
	if err != nil {
		if err == gcache.KeyNotFoundError {
			return false, nil
		}
		return false, err
	}

	// 将值反序列化到dst
	valBytes, err := json.Marshal(val)
	if err != nil {
		return false, err
	}

	return true, json.Unmarshal(valBytes, dst)
}

// MGet 批量从gcache获取值
func (g *GCacheStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	// 检查dstMap是否为*map[string]T类型
	mapValue := reflect.ValueOf(dstMap)
	if mapValue.Kind() != reflect.Ptr || mapValue.Elem().Kind() != reflect.Map {
		return errors.New("dstMap must be a pointer to a map")
	}

	// 创建结果map
	resultMap := make(map[string]interface{})
	for _, key := range keys {
		val, err := g.cache.Get(key)
		if err == nil {
			resultMap[key] = val
		}
	}

	// 将结果设置到dstMap
	return setMapValue(dstMap, resultMap)
}

// Exists 批量检查键存在性
func (g *GCacheStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool)
	for _, key := range keys {
		_, found := g.cache.Get(key)
		result[key] = found
	}
	return result, nil
}

// MSet 批量设置键值对，支持TTL
func (g *GCacheStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	for key, value := range items {
		if ttl > 0 {
			// 设置带TTL的键
			if err := g.cache.SetWithExpire(key, value, ttl); err != nil {
				return err
			}
		} else {
			// 设置永不过期的键
			if err := g.cache.Set(key, value); err != nil {
				return err
			}
		}
	}
	return nil
}

// Del 删除指定键
func (g *GCacheStore) Del(ctx context.Context, keys ...string) (int64, error) {
	var count int64
	for _, key := range keys {
		if err := g.cache.Remove(key); err == nil {
			count++
		}
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