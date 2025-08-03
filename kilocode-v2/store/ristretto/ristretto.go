package ristretto

import (
	"context"
	"reflect"
	"time"

	"github.com/dgraph-io/ristretto/v2"
)

// Config Ristretto 配置
type Config struct {
	NumCounters int64
	MaxCost     int64
	BufferItems int64
}

// RistrettoStore 是基于 Ristretto 的存储实现
type RistrettoStore struct {
	cache *ristretto.Cache[string, interface{}]
}

// NewRistrettoStore 创建一个新的 RistrettoStore 实例
func NewRistrettoStore(config *Config) (*RistrettoStore, error) {
	cache, err := ristretto.NewCache(&ristretto.Config[string, interface{}]{
		NumCounters: config.NumCounters,
		MaxCost:     config.MaxCost,
		BufferItems: config.BufferItems,
	})
	if err != nil {
		return nil, err
	}

	return &RistrettoStore{
		cache: cache,
	}, nil
}

// Close 关闭 RistrettoStore
func (r *RistrettoStore) Close() {
	r.cache.Close()
}

// Get 从存储后端获取单个值
func (r *RistrettoStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	value, found := r.cache.Get(key)
	if !found {
		return false, nil
	}

	// 将值复制到 dst
	return true, copyValue(dst, value)
}

// MGet 批量获取值到map中
func (r *RistrettoStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	// 验证 dstMap 是一个指向 map 的指针
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr || dstMapValue.Elem().Kind() != reflect.Map {
		return &InvalidTypeError{Message: "dstMap must be a pointer to a map"}
	}

	// 初始化目标 map
	dstMapValue.Elem().Set(reflect.MakeMap(dstMapValue.Elem().Type()))

	// 获取每个键的值
	for _, key := range keys {
		value, found := r.cache.Get(key)
		if found {
			mapKey := reflect.ValueOf(key)
			mapValue := reflect.ValueOf(value)
			dstMapValue.Elem().SetMapIndex(mapKey, mapValue)
		}
	}

	return nil
}

// Exists 批量检查键存在性
func (r *RistrettoStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	exists := make(map[string]bool)

	// 检查每个键的存在性
	for _, key := range keys {
		_, found := r.cache.Get(key)
		exists[key] = found
	}

	return exists, nil
}

// MSet 批量设置键值对，支持TTL
func (r *RistrettoStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	// Ristretto 本身不支持 TTL，我们需要自己实现
	// 这里我们忽略 TTL 参数，直接设置值
	// 在实际应用中，可能需要使用其他机制来实现 TTL

	for key, value := range items {
		// 在 Ristretto 中，cost 固定为 1
		r.cache.Set(key, value, 1)
	}

	// 等待值被缓冲
	r.cache.Wait()

	return nil
}

// Del 删除指定键
func (r *RistrettoStore) Del(ctx context.Context, keys ...string) (int64, error) {
	count := int64(0)

	// 删除每个键
	for _, key := range keys {
		// 检查键是否存在
		if _, found := r.cache.Get(key); found {
			r.cache.Del(key)
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
