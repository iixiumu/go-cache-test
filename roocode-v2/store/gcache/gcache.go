package gcache

import (
	"context"
	"reflect"
	"time"

	"go-cache/store"

	"github.com/bluele/gcache"
)

// GCacheStore 是基于gcache的Store接口实现
type GCacheStore struct {
	cache gcache.Cache
}

// NewGCacheStore 创建一个新的GCacheStore实例
func NewGCacheStore(cache gcache.Cache) store.Store {
	return &GCacheStore{
		cache: cache,
	}
}

// Get 从GCache获取单个值
func (g *GCacheStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	value, err := g.cache.Get(key)
	if err != nil {
		// 检查是否是键不存在的错误
		if err == gcache.KeyNotFoundError {
			return false, nil
		}
		return false, err
	}

	// 使用反射将值设置到dst
	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() != reflect.Ptr {
		return false, nil
	}
	dstValue = dstValue.Elem()

	valueReflect := reflect.ValueOf(value)
	if dstValue.Type() != valueReflect.Type() {
		return false, nil
	}

	dstValue.Set(valueReflect)
	return true, nil
}

// MGet 批量从GCache获取值到map中
func (g *GCacheStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	// 使用反射处理dstMap
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr {
		return nil // dstMap必须是指针
	}
	dstMapValue = dstMapValue.Elem()
	if dstMapValue.Kind() != reflect.Map {
		return nil // dstMap必须是指向map的指针
	}

	// 获取map的键和值类型
	mapType := dstMapValue.Type()
	keyType := mapType.Key()
	valueType := mapType.Elem()

	// 创建新的map
	newMap := reflect.MakeMap(mapType)

	// 遍历键
	for _, key := range keys {
		value, err := g.cache.Get(key)
		if err != nil {
			// 键不存在，跳过
			continue
		}

		// 检查值类型
		valueReflect := reflect.ValueOf(value)
		if valueReflect.Type() != valueType {
			continue
		}

		// 设置map中的值
		mapKey := reflect.ValueOf(key).Convert(keyType)
		newMap.SetMapIndex(mapKey, valueReflect)
	}

	// 设置dstMap的值
	dstMapValue.Set(newMap)
	return nil
}

// Exists 批量检查键存在性
func (g *GCacheStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	// 构建结果
	exists := make(map[string]bool)
	for _, key := range keys {
		_, err := g.cache.Get(key)
		exists[key] = err == nil
	}

	return exists, nil
}

// MSet 批量设置键值对，支持TTL
func (g *GCacheStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	// 批量设置
	for key, value := range items {
		if ttl > 0 {
			g.cache.SetWithExpire(key, value, ttl)
		} else {
			g.cache.Set(key, value)
		}
	}

	return nil
}

// Del 删除指定键
func (g *GCacheStore) Del(ctx context.Context, keys ...string) (int64, error) {
	count := int64(0)
	for _, key := range keys {
		if g.cache.Remove(key) {
			count++
		}
	}
	return count, nil
}
