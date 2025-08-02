package ristretto

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dgraph-io/ristretto"
	cache "go-cache"
)

// RistrettoStore Ristretto存储后端实现
type RistrettoStore struct {
	cache *ristretto.Cache
	mu    sync.RWMutex
}

// NewRistrettoStore 创建Ristretto存储实例
func NewRistrettoStore(config *ristretto.Config) (*RistrettoStore, error) {
	c, err := ristretto.NewCache(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create ristretto cache: %w", err)
	}

	return &RistrettoStore{
		cache: c,
	}, nil
}

// NewDefaultRistrettoStore 创建默认配置的Ristretto存储实例
func NewDefaultRistrettoStore() (*RistrettoStore, error) {
	config := &ristretto.Config{
		NumCounters: 1e7,     // 10M keys
		MaxCost:     1 << 30, // 1GB
		BufferItems: 64,
	}
	return NewRistrettoStore(config)
}

// Get 从Ristretto获取单个值
func (r *RistrettoStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	value, found := r.cache.Get(key)
	if !found {
		return false, nil
	}

	if strValue, ok := value.(string); ok {
		if err := cache.DeserializeValue(strValue, dst); err != nil {
			return false, fmt.Errorf("deserialize failed: %w", err)
		}
		return true, nil
	}

	return false, fmt.Errorf("invalid value type in cache: %T", value)
}

// MGet 批量获取值到map中
func (r *RistrettoStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	if len(keys) == 0 {
		return nil
	}

	if err := cache.ValidateMapPointer(dstMap); err != nil {
		return err
	}

	for _, key := range keys {
		value, found := r.cache.Get(key)
		if found {
			if strValue, ok := value.(string); ok {
				if err := cache.SetMapValue(dstMap, key, strValue); err != nil {
					return fmt.Errorf("set map value failed for key %s: %w", key, err)
				}
			}
		}
	}

	return nil
}

// Exists 批量检查键存在性
func (r *RistrettoStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool, len(keys))
	
	for _, key := range keys {
		_, found := r.cache.Get(key)
		result[key] = found
	}

	return result, nil
}

// MSet 批量设置键值对，支持TTL
func (r *RistrettoStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	for key, value := range items {
		serialized, err := cache.SerializeValue(value)
		if err != nil {
			return fmt.Errorf("serialize failed for key %s: %w", key, err)
		}

		cost := int64(len(serialized))
		if cost == 0 {
			cost = 1
		}

		success := r.cache.SetWithTTL(key, serialized, cost, ttl)
		if !success {
			return fmt.Errorf("failed to set key %s in ristretto cache", key)
		}
	}

	// 等待所有设置操作完成
	r.cache.Wait()
	return nil
}

// Del 删除指定键
func (r *RistrettoStore) Del(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	var deleted int64
	for _, key := range keys {
		// 检查键是否存在
		if _, found := r.cache.Get(key); found {
			r.cache.Del(key)
			deleted++
		}
	}

	return deleted, nil
}

// Close 关闭缓存
func (r *RistrettoStore) Close() {
	r.cache.Close()
}