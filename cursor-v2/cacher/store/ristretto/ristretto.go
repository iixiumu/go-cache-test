package ristretto

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"go-cache/cacher/store"

	"github.com/dgraph-io/ristretto/v2"
)

// RistrettoStore 基于Ristretto的Store实现
type RistrettoStore struct {
	cache *ristretto.Cache[string, interface{}]
}

// NewRistrettoStore 创建新的Ristretto Store
func NewRistrettoStore(cache *ristretto.Cache[string, interface{}]) (store.Store, error) {
	if cache == nil {
		return nil, fmt.Errorf("cache cannot be nil")
	}

	return &RistrettoStore{cache: cache}, nil
}

// Get 从Ristretto获取单个值
func (r *RistrettoStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	value, found := r.cache.Get(key)
	if !found {
		return false, nil
	}

	// 直接复制值到目标
	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() != reflect.Ptr {
		return false, fmt.Errorf("dst must be a pointer")
	}

	srcValue := reflect.ValueOf(value)
	dstElem := dstValue.Elem()

	// 如果类型匹配，直接复制
	if srcValue.Type() == dstElem.Type() {
		dstElem.Set(srcValue)
		return true, nil
	}

	// 如果类型不匹配，尝试类型转换
	if srcValue.CanConvert(dstElem.Type()) {
		dstElem.Set(srcValue.Convert(dstElem.Type()))
		return true, nil
	}

	return false, fmt.Errorf("cannot convert %T to %T", value, dst)
}

// MGet 批量获取值
func (r *RistrettoStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	if len(keys) == 0 {
		return nil
	}

	// 使用反射设置dstMap
	dstValue := reflect.ValueOf(dstMap)
	if dstValue.Kind() != reflect.Ptr || dstValue.Elem().Kind() != reflect.Map {
		return fmt.Errorf("dstMap must be a pointer to map")
	}

	dstMapValue := dstValue.Elem()
	result := reflect.MakeMap(dstMapValue.Type())

	for _, key := range keys {
		value, found := r.cache.Get(key)
		if found {
			result.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
		}
	}

	dstMapValue.Set(result)
	return nil
}

// Exists 检查键存在性
func (r *RistrettoStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	if len(keys) == 0 {
		return make(map[string]bool), nil
	}

	result := make(map[string]bool)
	for _, key := range keys {
		_, found := r.cache.Get(key)
		result[key] = found
	}

	return result, nil
}

// MSet 批量设置键值对
func (r *RistrettoStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	// 计算成本（这里简单使用1作为默认成本）
	cost := int64(1)

	for key, value := range items {
		// 使用SetWithTTL方法支持TTL
		if ttl > 0 {
			r.cache.SetWithTTL(key, value, cost, ttl)
		} else {
			r.cache.Set(key, value, cost)
		}
	}

	// 等待值通过缓冲区
	r.cache.Wait()

	return nil
}

// Del 删除指定键
func (r *RistrettoStore) Del(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	deleted := int64(0)
	for _, key := range keys {
		r.cache.Del(key)
		deleted++
	}

	return deleted, nil
}
