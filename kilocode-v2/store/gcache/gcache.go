package gcache

import (
	"context"
	"reflect"
	"time"

	"github.com/bluele/gcache"
)

// Config GCache 配置
type Config struct {
	Size int
}

// GCacheStore 是基于 GCache 的存储实现
type GCacheStore struct {
	cache gcache.Cache
}

// NewGCacheStore 创建一个新的 GCacheStore 实例
func NewGCacheStore(config *Config) (*GCacheStore, error) {
	cache := gcache.New(config.Size).Build()

	return &GCacheStore{
		cache: cache,
	}, nil
}

// Close 关闭 GCacheStore
func (g *GCacheStore) Close() {
	// GCache 没有显式的关闭方法
}

// Get 从存储后端获取单个值
func (g *GCacheStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	value, err := g.cache.Get(key)
	if err != nil {
		// GCache 在键不存在时返回错误
		return false, nil
	}

	// 将值复制到 dst
	return true, copyValue(dst, value)
}

// MGet 批量获取值到map中
func (g *GCacheStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	// 验证 dstMap 是一个指向 map 的指针
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr || dstMapValue.Elem().Kind() != reflect.Map {
		return &InvalidTypeError{Message: "dstMap must be a pointer to a map"}
	}

	// 初始化目标 map
	dstMapValue.Elem().Set(reflect.MakeMap(dstMapValue.Elem().Type()))

	// 获取每个键的值
	for _, key := range keys {
		value, err := g.cache.Get(key)
		if err == nil {
			mapKey := reflect.ValueOf(key)
			mapValue := reflect.ValueOf(value)
			dstMapValue.Elem().SetMapIndex(mapKey, mapValue)
		}
	}

	return nil
}

// Exists 批量检查键存在性
func (g *GCacheStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	exists := make(map[string]bool)

	// 检查每个键的存在性
	for _, key := range keys {
		_, err := g.cache.Get(key)
		exists[key] = err == nil
	}

	return exists, nil
}

// MSet 批量设置键值对，支持TTL
func (g *GCacheStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
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

	// 删除每个键
	for _, key := range keys {
		// 检查键是否存在
		if _, err := g.cache.Get(key); err == nil {
			g.cache.Remove(key)
			count++
		}
	}

	return count, nil
}

// copyValue 将 src 的值复制到 dst
func copyValue(dst interface{}, src interface{}) error {
	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() != reflect.Ptr {
		return &InvalidTypeError{Message: "dst must be a pointer"}
	}

	srcValue := reflect.ValueOf(src)
	dstValue.Elem().Set(srcValue)

	return nil
}

// InvalidTypeError 无效类型错误
type InvalidTypeError struct {
	Message string
}

func (e *InvalidTypeError) Error() string {
	return e.Message
}
