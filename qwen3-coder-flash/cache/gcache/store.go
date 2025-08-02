package gcache

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/xiumu/go-cache/cache"
)

// GCacheStore 实现Store接口，基于简单的 in-memory cache for testing
type GCacheStore struct {
	data map[string]interface{}
	mutex sync.RWMutex
}

// NewGCacheStore 创建新的GCacheStore实例
func NewGCacheStore() *GCacheStore {
	return &GCacheStore{
		data: make(map[string]interface{}),
	}
}

// Get 从内存获取单个值
func (g *GCacheStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	
	if val, exists := g.data[key]; exists {
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
	
	return false, nil
}

// MGet 批量获取值到map中
func (g *GCacheStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	
	// 使用反射来处理不同的map类型
	mapValue := reflect.ValueOf(dstMap)
	if mapValue.Kind() != reflect.Ptr || mapValue.Elem().Kind() != reflect.Map {
		return fmt.Errorf("dstMap must be a pointer to a map")
	}

	mapElem := mapValue.Elem()
	
	// Process each key individually
	for _, key := range keys {
		if val, exists := g.data[key]; exists {
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
func (g *GCacheStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	
	result := make(map[string]bool)
	
	for _, key := range keys {
		_, exists := g.data[key]
		result[key] = exists
	}
	
	return result, nil
}

// MSet 批量设置键值对，支持TTL
func (g *GCacheStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	
	for key, value := range items {
		g.data[key] = value
	}
	
	return nil
}

// Del 删除指定键
func (g *GCacheStore) Del(ctx context.Context, keys ...string) (int64, error) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	
	count := int64(0)
	
	for _, key := range keys {
		if _, exists := g.data[key]; exists {
			delete(g.data, key)
			count++
		}
	}
	
	return count, nil
}