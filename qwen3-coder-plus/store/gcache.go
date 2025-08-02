package store

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/bluele/gcache"
	"github.com/xiumu/go-cache/cache"
)

// gcacheStore GCache存储实现
type gcacheStore struct {
	cache gcache.Cache
}

// NewGCacheStore 创建一个新的GCache存储实例
func NewGCacheStore(cache gcache.Cache) cache.Store {
	return &gcacheStore{
		cache: cache,
	}
}

// Get 从存储后端获取单个值
func (g *gcacheStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	// 从GCache中获取值
	val, err := g.cache.Get(key)
	if err != nil {
		// 检查是否是未找到的错误
		if err.Error() == "not found" {
			return false, nil
		}
		return false, err
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
func (g *gcacheStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
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
		// 从GCache中获取值
		val, err := g.cache.Get(key)
		if err != nil {
			// 跳过未找到的键
			if err.Error() == "not found" {
				continue
			}
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
		err = json.Unmarshal(data, elem)
		if err != nil {
			continue
		}

		// 将值设置到map中
		dstMapValue.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(elem).Elem())
	}

	return nil
}

// Exists 批量检查键存在性
func (g *gcacheStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool)

	// 遍历所有键
	for _, key := range keys {
		// 检查键是否存在
		_, err := g.cache.Get(key)
		result[key] = err == nil
	}

	return result, nil
}

// MSet 批量设置键值对，支持TTL
func (g *gcacheStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	// 遍历所有键值对
	for key, value := range items {
		// 序列化值
		data, err := json.Marshal(value)
		if err != nil {
			continue
		}

		// 根据是否有TTL设置键值对
		if ttl > 0 {
			g.cache.SetWithExpire(key, data, ttl)
		} else {
			g.cache.Set(key, data)
		}
	}

	return nil
}

// Del 删除指定键
func (g *gcacheStore) Del(ctx context.Context, keys ...string) (int64, error) {
	var deleted int64

	// 遍历所有键
	for _, key := range keys {
		// 从缓存中删除键
		if g.cache.Remove(key) {
			deleted++
		}
	}

	return deleted, nil
}