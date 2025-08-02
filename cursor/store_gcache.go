package cache

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/bluele/gcache"
)

// GCacheStore 基于GCache的Store实现
type GCacheStore struct {
	cache gcache.Cache
}

// NewGCacheStore 创建新的GCache存储实例
func NewGCacheStore(cache gcache.Cache) Store {
	return &GCacheStore{
		cache: cache,
	}
}

// Get 从GCache获取单个值
func (g *GCacheStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	value, err := g.cache.Get(key)
	if err != nil {
		if err == gcache.KeyNotFoundError {
			return false, nil
		}
		return false, err
	}

	// 尝试JSON反序列化
	if jsonStr, ok := value.(string); ok {
		if err := json.Unmarshal([]byte(jsonStr), dst); err != nil {
			return false, err
		}
		return true, nil
	}

	// 如果值不是字符串，尝试直接设置
	if err := setValue(dst, value); err != nil {
		return false, err
	}

	return true, nil
}

// MGet 批量获取值
func (g *GCacheStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	if len(keys) == 0 {
		return nil
	}

	// 验证dstMap类型
	dstValue := reflect.ValueOf(dstMap)
	if dstValue.Kind() != reflect.Ptr || dstValue.Elem().Kind() != reflect.Map {
		return ErrInvalidDstMap
	}

	dstMapValue := dstValue.Elem()
	dstMapType := dstMapValue.Type()

	// 确保map已初始化
	if dstMapValue.IsNil() {
		dstMapValue.Set(reflect.MakeMap(dstMapType))
	}

	// 批量获取
	for _, key := range keys {
		value, err := g.cache.Get(key)
		if err == nil {
			// 创建目标类型的零值
			elemType := dstMapType.Elem()
			elemValue := reflect.New(elemType).Elem()

			// 尝试JSON反序列化
			if jsonStr, ok := value.(string); ok {
				if err := json.Unmarshal([]byte(jsonStr), elemValue.Addr().Interface()); err == nil {
					dstMapValue.SetMapIndex(reflect.ValueOf(key), elemValue)
				}
			} else {
				// 尝试直接设置
				if err := setValue(elemValue.Addr().Interface(), value); err == nil {
					dstMapValue.SetMapIndex(reflect.ValueOf(key), elemValue)
				}
			}
		}
	}

	return nil
}

// Exists 批量检查键存在性
func (g *GCacheStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool)

	for _, key := range keys {
		_, err := g.cache.Get(key)
		result[key] = err == nil
	}

	return result, nil
}

// MSet 批量设置键值对
func (g *GCacheStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	for key, value := range items {
		// 将值序列化为JSON字符串
		jsonBytes, err := json.Marshal(value)
		if err != nil {
			return err
		}

		// GCache不支持TTL，所以忽略ttl参数
		g.cache.Set(key, string(jsonBytes))
	}

	return nil
}

// Del 删除指定键
func (g *GCacheStore) Del(ctx context.Context, keys ...string) (int64, error) {
	deleted := int64(0)

	for _, key := range keys {
		if g.cache.Remove(key) {
			deleted++
		}
	}

	return deleted, nil
}
