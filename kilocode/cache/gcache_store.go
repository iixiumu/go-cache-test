package cache

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/bluele/gcache"
)

// gcacheStore 实现了Store接口，使用GCache作为存储后端
type gcacheStore struct {
	cache gcache.Cache
}

// NewGCacheStore 创建一个新的GCache Store实例
func NewGCacheStore(cache gcache.Cache) Store {
	return &gcacheStore{
		cache: cache,
	}
}

// Get 从GCache获取单个值
func (g *gcacheStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	value, err := g.cache.Get(key)
	if err != nil {
		// 如果键不存在，返回false
		if err == gcache.KeyNotFoundError {
			return false, nil
		}
		return false, err
	}

	// 将值序列化为字节
	valBytes, ok := value.([]byte)
	if !ok {
		return false, nil
	}

	// 反序列化值到dst
	err = json.Unmarshal(valBytes, dst)
	if err != nil {
		return false, err
	}

	return true, nil
}

// MGet 批量从GCache获取值
func (g *gcacheStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
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
		value, err := g.cache.Get(key)
		if err != nil {
			// 如果键不存在，跳过
			if err == gcache.KeyNotFoundError {
				continue
			}
			return err
		}

		// 将值序列化为字节
		valBytes, ok := value.([]byte)
		if !ok {
			continue
		}

		// 创建对应类型的值
		valuePtr := reflect.New(mapValueType).Interface()

		// 反序列化
		err = json.Unmarshal(valBytes, valuePtr)
		if err != nil {
			continue
		}

		// 设置map值
		dstMapElem.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(valuePtr).Elem())
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
		valBytes, err := json.Marshal(value)
		if err != nil {
			return err
		}

		// 设置值到缓存
		if ttl > 0 {
			err = g.cache.SetWithExpire(key, valBytes, ttl)
		} else {
			err = g.cache.Set(key, valBytes)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

// Del 从GCache删除指定键
func (g *gcacheStore) Del(ctx context.Context, keys ...string) (int64, error) {
	var deleted int64
	for _, key := range keys {
		// GCache的Remove方法没有返回错误，总是返回nil
		// 我们无法准确知道是否真的删除了键，所以简单地认为每个键都被删除了
		g.cache.Remove(key)
		deleted++
	}
	return deleted, nil
}
