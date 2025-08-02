package ristretto

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/xiumu/go-cache/cache"
)

// RistrettoStore 实现Store接口，基于Ristretto
type RistrettoStore struct {
	cache *ristretto.Cache
	mutex sync.RWMutex
}

// NewRistrettoStore 创建新的RistrettoStore实例
func NewRistrettoStore(config *ristretto.Config) (*RistrettoStore, error) {
	cache, err := ristretto.NewCache(config)
	if err != nil {
		return nil, err
	}
	
	return &RistrettoStore{
		cache: cache,
	}, nil
}

// Get 从Ristretto获取单个值
func (r *RistrettoStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	val, found := r.cache.Get(key)
	if !found {
		return false, nil
	}
	
	// Convert to byte slice if needed
	var bytes []byte
	switch v := val.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		// Try to marshal it
		b, err := json.Marshal(v)
		if err != nil {
			return false, err
		}
		bytes = b
	}
	
	err := json.Unmarshal(bytes, dst)
	if err != nil {
		return false, err
	}
	
	return true, nil
}

// MGet 批量获取值到map中
func (r *RistrettoStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	// 使用反射来处理不同的map类型
	mapValue := reflect.ValueOf(dstMap)
	if mapValue.Kind() != reflect.Ptr || mapValue.Elem().Kind() != reflect.Map {
		return fmt.Errorf("dstMap must be a pointer to a map")
	}

	mapElem := mapValue.Elem()
	
	// Process each key individually
	for _, key := range keys {
		val, found := r.cache.Get(key)
		if found {
			// Convert to byte slice if needed
			var bytes []byte
			switch v := val.(type) {
			case string:
				bytes = []byte(v)
			case []byte:
				bytes = v
			default:
				// Try to marshal it
				b, err := json.Marshal(v)
				if err != nil {
					return err
				}
				bytes = b
			}
			
			var result interface{}
			if err := json.Unmarshal(bytes, &result); err != nil {
				return err
			}
			mapElem.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(result))
		}
	}

	return nil
}

// Exists 批量检查键存在性
func (r *RistrettoStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool)
	
	for _, key := range keys {
		_, found := r.cache.Get(key)
		result[key] = found
	}
	
	return result, nil
}

// MSet 批量设置键值对，支持TTL
func (r *RistrettoStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	// Ristretto doesn't support TTL per item, so we'll ignore TTL for now
	for key, value := range items {
		r.cache.Set(key, value, 1)
	}
	
	return nil
}

// Del 删除指定键
func (r *RistrettoStore) Del(ctx context.Context, keys ...string) (int64, error) {
	count := int64(0)
	
	for _, key := range keys {
		if r.cache.Del(key) {
			count++
		}
	}
	
	return count, nil
}