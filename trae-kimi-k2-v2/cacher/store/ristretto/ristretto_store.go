package ristretto

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/dgraph-io/ristretto/v2"
)

// RistrettoStore Ristretto内存存储实现
type RistrettoStore struct {
	cache *ristretto.Cache[string, interface{}]
}

// NewRistrettoStore 创建Ristretto存储实例
func NewRistrettoStore() (*RistrettoStore, error) {
	cache, err := ristretto.NewCache(&ristretto.Config[string, interface{}]{
		NumCounters: 1e7,     // 用于跟踪访问频率的计数器数量
		MaxCost:     1 << 30, // 最大缓存大小 (1GB)
		BufferItems: 64,      // 每个Get缓冲区的键值对数量
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create ristretto cache: %w", err)
	}

	return &RistrettoStore{
		cache: cache,
	}, nil
}

// Get 从Ristretto获取单个值
func (r *RistrettoStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	value, found := r.cache.Get(key)
	if !found {
		return false, nil
	}

	if dst == nil {
		return true, nil
	}

	// 使用反射设置值
	if err := setValue(dst, value); err != nil {
		return false, fmt.Errorf("failed to set value: %w", err)
	}

	return true, nil
}

// MGet 批量获取值到map中
func (r *RistrettoStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	if len(keys) == 0 {
		return nil
	}

	result := make(map[string]interface{})
	for _, key := range keys {
		if value, found := r.cache.Get(key); found {
			result[key] = value
		}
	}

	// 使用反射设置目标map
	return setMapValue(dstMap, result)
}

// Exists 批量检查键存在性
func (r *RistrettoStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	results := make(map[string]bool)
	for _, key := range keys {
		_, found := r.cache.Get(key)
		results[key] = found
	}
	return results, nil
}

// MSet 批量设置键值对，支持TTL
func (r *RistrettoStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	for key, value := range items {
		cost := int64(1) // 默认成本
		
		// 计算值的近似大小作为成本
		if str, ok := value.(string); ok {
			cost = int64(len(str))
		}

		if ttl > 0 {
			r.cache.SetWithTTL(key, value, cost, ttl)
		} else {
			r.cache.Set(key, value, cost)
		}
	}

	// 等待Ristretto处理缓存
	r.cache.Wait()

	return nil
}

// Del 删除指定键
func (r *RistrettoStore) Del(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	var deleted int64 = 0
	for _, key := range keys {
		r.cache.Del(key)
		deleted++
	}
	return deleted, nil
}

// Close 关闭Ristretto缓存
func (r *RistrettoStore) Close() error {
	r.cache.Close()
	return nil
}

// setValue 使用反射设置值
func setValue(dst interface{}, value interface{}) error {
	if dst == nil {
		return fmt.Errorf("destination is nil")
	}

	// 获取目标值的反射
	ptr := reflect.ValueOf(dst)
	if ptr.Kind() != reflect.Ptr {
		return fmt.Errorf("destination must be a pointer")
	}

	// 解引用指针
	val := ptr.Elem()
	if !val.CanSet() {
		return fmt.Errorf("cannot set destination value")
	}

	// 设置值
	val.Set(reflect.ValueOf(value))
	return nil
}

// setMapValue 使用反射设置map值
func setMapValue(dstMap interface{}, values map[string]interface{}) error {
	if dstMap == nil {
		return fmt.Errorf("destination map is nil")
	}

	ptr := reflect.ValueOf(dstMap)
	if ptr.Kind() != reflect.Ptr {
		return fmt.Errorf("destination must be a pointer to map")
	}

	mapVal := ptr.Elem()
	if mapVal.Kind() != reflect.Map {
		return fmt.Errorf("destination must be a pointer to map[string]interface{}")
	}

	for k, v := range values {
		mapVal.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v))
	}

	return nil
}