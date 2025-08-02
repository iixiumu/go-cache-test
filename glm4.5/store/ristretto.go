package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/dgraph-io/ristretto"
	"go-cache/internal"
)

// RistrettoStore Ristretto实现的Store接口
type RistrettoStore struct {
	cache *ristretto.Cache
}

// NewRistrettoStore 创建Ristretto存储实例
func NewRistrettoStore(maxItems int64, maxCost int64) (*RistrettoStore, error) {
	config := &ristretto.Config{
		NumCounters: maxItems * 10,
		MaxCost:     maxCost,
		BufferItems: 64,
	}

	cache, err := ristretto.NewCache(config)
	if err != nil {
		return nil, err
	}

	return &RistrettoStore{cache: cache}, nil
}

// Get 从Ristretto获取单个值
func (r *RistrettoStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	value, found := r.cache.Get(key)
	if !found {
		return false, nil
	}

	data, ok := value.([]byte)
	if !ok {
		return false, nil
	}

	if err := json.Unmarshal(data, dst); err != nil {
		return false, err
	}

	return true, nil
}

// MGet 批量获取值到map中
func (r *RistrettoStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	if len(keys) == 0 {
		return nil
	}

	keyType, valueType, err := internal.GetTypeOfMap(dstMap)
	if err != nil {
		return err
	}

	for _, key := range keys {
		value, found := r.cache.Get(key)
		if !found {
			continue
		}

		data, ok := value.([]byte)
		if !ok {
			continue
		}

		var parsedValue interface{}
		if err := json.Unmarshal(data, &parsedValue); err != nil {
			continue
		}

		if err := internal.SetMapValueWithType(dstMap, key, parsedValue, keyType, valueType); err != nil {
			return err
		}
	}

	return nil
}

// Exists 批量检查键存在性
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

// MSet 批量设置键值对，支持TTL
func (r *RistrettoStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	for key, value := range items {
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}

		cost := int64(len(data))
		if cost == 0 {
			cost = 1
		}

		if ttl > 0 {
			r.cache.SetWithTTL(key, data, cost, ttl)
		} else {
			r.cache.Set(key, data, cost)
		}
	}

	return nil
}

// Del 删除指定键
func (r *RistrettoStore) Del(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	var deleted int64
	for _, key := range keys {
		r.cache.Del(key)
		deleted++
	}

	return deleted, nil
}

// Clear 清空缓存
func (r *RistrettoStore) Clear() {
	r.cache.Clear()
}

// Close 关闭缓存
func (r *RistrettoStore) Close() {
	r.cache.Close()
}

// RistrettoStoreOptions Ristretto存储选项
type RistrettoStoreOptions struct {
	MaxItems int64
	MaxCost  int64
}

// NewRistrettoStoreWithOptions 使用选项创建Ristretto存储实例
func NewRistrettoStoreWithOptions(opts RistrettoStoreOptions) (*RistrettoStore, error) {
	return NewRistrettoStore(opts.MaxItems, opts.MaxCost)
}

// NewDefaultRistrettoStore 创建默认配置的Ristretto存储实例
func NewDefaultRistrettoStore() (*RistrettoStore, error) {
	return NewRistrettoStore(10000, 1000000)
}