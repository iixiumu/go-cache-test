package store

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/bluele/gcache"
)

// gcacheStore 是基于GCache的Store实现
type gcacheStore struct {
	cache gcache.Cache
}

// NewGCacheStore 创建一个新的GCache存储实例
func NewGCacheStore(cache gcache.Cache) Store {
	return &gcacheStore{
		cache: cache,
	}
}

// Get 从GCache获取单个值
func (g *gcacheStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	val, err := g.cache.Get(key)
	if err != nil {
		// GCache返回特定的错误来表示键不存在
		if err == gcache.KeyNotFoundError {
			return false, nil
		}
		return false, err
	}

	// val是字节切片，需要反序列化
	data, ok := val.([]byte)
	if !ok {
		return false, nil
	}

	// 反序列化到dst
	err = json.Unmarshal(data, dst)
	if err != nil {
		return false, err
	}

	return true, nil
}

// MGet 批量从GCache获取值
func (g *gcacheStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
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
		val, err := g.cache.Get(key)
		if err != nil {
			// GCache返回特定的错误来表示键不存在
			if err == gcache.KeyNotFoundError {
				continue
			}
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
		err = json.Unmarshal(data, mapValue)
		if err != nil {
			continue
		}

		// 设置map元素
		dstMapElem.SetMapIndex(mapKey, reflect.ValueOf(mapValue).Elem())
	}

	return nil
}

// Exists 批量检查键在GCache中的存在性
func (g *gcacheStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool)

	for _, key := range keys {
		_, err := g.cache.Get(key)
		result[key] = err == nil
	}

	return result, nil
}

// MSet 批量设置键值对到GCache
func (g *gcacheStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	for key, value := range items {
		// 序列化值
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}

		// 设置到缓存中
		if ttl > 0 {
			err = g.cache.SetWithExpire(key, data, ttl)
		} else {
			err = g.cache.Set(key, data)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

// Del 从GCache删除指定键
func (g *gcacheStore) Del(ctx context.Context, keys ...string) (int64, error) {
	count := int64(0)

	for _, key := range keys {
		// GCache的Remove方法返回bool表示是否成功删除
		if g.cache.Remove(key) {
			count++
		}
	}

	return count, nil
}
