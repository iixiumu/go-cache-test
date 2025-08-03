package gcache

import (
	"context"
	"reflect"
	"sync"
	"time"

	"github.com/bluele/gcache"
)

// gcacheStore 是 gcache 实现的 Store
type gcacheStore struct {
	cache gcache.Cache
	mutex sync.RWMutex
}

// NewGCacheStore 创建一个新的 gcache Store 实例
func NewGCacheStore(cache gcache.Cache) Store {
	return &gcacheStore{
		cache: cache,
	}
}

// Get 从 gcache 获取单个值
func (g *gcacheStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	value, err := g.cache.Get(key)
	if err != nil {
		if err == gcache.KeyNotFoundError {
			return false, nil
		}
		return false, err
	}

	// 直接赋值给dst（因为是内存存储，不需要序列化）
	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() == reflect.Ptr && !dstValue.IsNil() {
		dstElem := dstValue.Elem()
		valueReflect := reflect.ValueOf(value)
		if valueReflect.Type().AssignableTo(dstElem.Type()) {
			dstElem.Set(valueReflect)
		}
	}

	return true, nil
}

// MGet 批量获取值到map中
func (g *gcacheStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	// 将结果填充到dstMap中
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr || dstMapValue.IsNil() {
		return nil
	}

	mapValue := dstMapValue.Elem()

	// 假设dstMap是一个map[string]interface{}类型
	if mapValue.Kind() == reflect.Map && mapValue.Type().Key().Kind() == reflect.String {
		for _, key := range keys {
			value, err := g.cache.Get(key)
			if err == nil {
				mapValue.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
			}
		}
	}

	return nil
}

// Exists 批量检查键存在性
func (g *gcacheStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	result := make(map[string]bool)
	for _, key := range keys {
		_, err := g.cache.Get(key)
		result[key] = err == nil
	}
	return result, nil
}

// MSet 批量设置键值对，支持TTL
func (g *gcacheStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	for key, value := range items {
		// 直接存储值（内存存储，不需要序列化）
		if ttl > 0 {
			err := g.cache.SetWithExpire(key, value, ttl)
			if err != nil {
				return err
			}
		} else {
			err := g.cache.Set(key, value)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Del 删除指定键
func (g *gcacheStore) Del(ctx context.Context, keys ...string) (int64, error) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	count := int64(0)
	for _, key := range keys {
		err := g.cache.Remove(key)
		if err == nil {
			count++
		}
	}

	return count, nil
}
