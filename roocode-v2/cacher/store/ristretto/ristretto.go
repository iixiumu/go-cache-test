package ristretto

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"go-cache/cacher/store"

	ristretto "github.com/dgraph-io/ristretto/v2"
)

// RistrettoStore 实现了Store接口的Ristretto内存存储
type RistrettoStore struct {
	cache *ristretto.Cache[string, interface{}]
	mu    sync.RWMutex
	ttl   map[string]time.Time
}

// NewRistrettoStore 创建一个新的RistrettoStore实例
func NewRistrettoStore() (store.Store, error) {
	cache, err := ristretto.NewCache(&ristretto.Config[string, interface{}]{
		NumCounters: 1e7,     // number of keys to track frequency of (10M).
		MaxCost:     1 << 30, // maximum cost of cache (1GB).
		BufferItems: 64,      // number of keys per Get buffer.
	})
	if err != nil {
		return nil, err
	}

	return &RistrettoStore{
		cache: cache,
		ttl:   make(map[string]time.Time),
	}, nil
}

// Get 从Ristretto获取单个值
func (r *RistrettoStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 检查TTL
	if ttl, exists := r.ttl[key]; exists {
		if time.Now().After(ttl) {
			// 过期，从缓存中删除
			r.cache.Del(key)
			delete(r.ttl, key)
			return false, nil
		}
	}

	// 从缓存获取值
	value, found := r.cache.Get(key)
	if !found {
		return false, nil
	}

	// 使用反射将值复制到目标变量
	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() != reflect.Ptr {
		return false, fmt.Errorf("dst must be a pointer")
	}

	// 获取目标变量的类型
	dstElem := dstValue.Elem()
	valueReflect := reflect.ValueOf(value)

	// 检查类型兼容性
	if !valueReflect.Type().AssignableTo(dstElem.Type()) {
		// 尝试转换类型
		if valueReflect.CanConvert(dstElem.Type()) {
			dstElem.Set(valueReflect.Convert(dstElem.Type()))
		} else {
			return false, fmt.Errorf("cannot assign %T to %T", value, dst)
		}
	} else {
		dstElem.Set(valueReflect)
	}

	return true, nil
}

// MGet 批量从Ristretto获取值
func (r *RistrettoStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 使用反射来处理目标map
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr || dstMapValue.Elem().Kind() != reflect.Map {
		return fmt.Errorf("dstMap must be a pointer to a map")
	}

	// 获取map的类型信息
	mapType := dstMapValue.Elem().Type()
	keyType := mapType.Key()
	valueType := mapType.Elem()

	// 创建新的map
	newMap := reflect.MakeMap(mapType)

	// 处理每个键
	for _, key := range keys {
		// 检查TTL
		if ttl, exists := r.ttl[key]; exists {
			if time.Now().After(ttl) {
				// 过期，从缓存中删除
				r.cache.Del(key)
				delete(r.ttl, key)
				continue
			}
		}

		// 从缓存获取值
		value, found := r.cache.Get(key)
		if found {
			// 创建对应类型的值
			valueReflect := reflect.ValueOf(value)

			// 检查类型兼容性
			if valueReflect.Type().AssignableTo(valueType) {
				// 将键值对添加到map中
				newMap.SetMapIndex(reflect.ValueOf(key).Convert(keyType), valueReflect)
			} else if valueReflect.CanConvert(valueType) {
				// 尝试转换类型
				convertedValue := valueReflect.Convert(valueType)
				newMap.SetMapIndex(reflect.ValueOf(key).Convert(keyType), convertedValue)
			}
		}
	}

	// 将结果复制到目标map
	dstMapValue.Elem().Set(newMap)

	return nil
}

// Exists 批量检查键在Ristretto中的存在性
func (r *RistrettoStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]bool)

	for _, key := range keys {
		// 检查TTL
		if ttl, exists := r.ttl[key]; exists {
			if time.Now().After(ttl) {
				// 过期，从缓存中删除
				r.cache.Del(key)
				delete(r.ttl, key)
				result[key] = false
				continue
			}
		}

		// 检查缓存中是否存在
		_, found := r.cache.Get(key)
		result[key] = found
	}

	return result, nil
}

// MSet 批量设置键值对到Ristretto
func (r *RistrettoStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 设置每个键值对
	for key, value := range items {
		// 设置值到缓存中，成本为1
		r.cache.Set(key, value, 1)

		// 如果设置了TTL，记录过期时间
		if ttl > 0 {
			r.ttl[key] = time.Now().Add(ttl)
		} else {
			// 如果没有TTL，删除之前的TTL记录
			delete(r.ttl, key)
		}
	}

	// 等待值通过缓冲区
	r.cache.Wait()

	return nil
}

// Del 从Ristretto删除指定键
func (r *RistrettoStore) Del(ctx context.Context, keys ...string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var deleted int64
	for _, key := range keys {
		// 检查键是否存在
		if _, found := r.cache.Get(key); found {
			// 从缓存中删除
			r.cache.Del(key)
			deleted++
		}

		// 删除TTL记录
		delete(r.ttl, key)
	}

	return deleted, nil
}
