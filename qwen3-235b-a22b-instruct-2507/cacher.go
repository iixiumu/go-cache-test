package cache

import (
	"context"
	"reflect"
	"time"
)

// Store 底层存储接口，提供基础的键值存储操作
type Store interface {
	// Get 从存储后端获取单个值
	// key: 键名
	// dst: 目标变量的指针，用于接收反序列化后的值
	// 返回: 是否找到该键, 错误信息
	Get(ctx context.Context, key string, dst interface{}) (bool, error)

	// MGet 批量获取值到map中
	// keys: 要获取的键列表
	// dstMap: 目标map的指针，用于接收结果，类型为*map[string]T
	// 返回: 错误信息
	MGet(ctx context.Context, keys []string, dstMap interface{}) error

	// Exists 批量检查键存在性
	// keys: 要检查的键列表
	// 返回: map[string]bool 键存在性映射, 错误信息
	Exists(ctx context.Context, keys []string) (map[string]bool, error)

	// MSet 批量设置键值对，支持TTL
	// items: 键值对映射
	// ttl: 过期时间，0表示永不过期
	// 返回: 错误信息
	MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error

	// Del 删除指定键
	// keys: 要删除的键列表
	// 返回: 实际删除的键数量, 错误信息
	Del(ctx context.Context, keys ...string) (int64, error)
}

// FallbackFunc 回退函数类型
// 当缓存未命中时执行，用于从数据源获取数据
// key: 请求的键
// 返回: 获取到的值, 是否找到, 错误信息
type FallbackFunc func(ctx context.Context, key string) (interface{}, bool, error)

// BatchFallbackFunc 批量回退函数类型
// 当批量缓存部分未命中时执行，用于从数据源批量获取数据
// keys: 未命中的键列表
// 返回: 键值映射, 错误信息
type BatchFallbackFunc func(ctx context.Context, keys []string) (map[string]interface{}, error)

// CacheOptions 缓存选项
type CacheOptions struct {
	// TTL 缓存过期时间，0表示永不过期
	TTL time.Duration
}

// Cacher 高级缓存接口，提供带回退机制的缓存操作
type Cacher interface {
	// Get 获取单个缓存项，缓存未命中时执行回退函数并缓存结果
	// key: 键名
	// dst: 目标变量的指针，用于接收值
	// fallback: 缓存未命中时的回退函数
	// opts: 缓存选项，可以为nil使用默认选项
	// 返回: 是否找到值（包括从回退函数获取）, 错误信息
	Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error)

	// MGet 批量获取缓存项，支持部分命中和批量回退
	// keys: 要获取的键列表
	// dstMap: 目标map的指针，用于接收结果，类型为*map[string]T
	// fallback: 批量回退函数，处理未命中的键
	// opts: 缓存选项，可以为nil使用默认选项
	// 返回: 错误信息
	MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error

	// MDelete 批量清除缓存项
	// keys: 要删除的键列表
	// 返回: 实际删除的键数量, 错误信息
	MDelete(ctx context.Context, keys []string) (int64, error)

	// MRefresh 批量强制刷新缓存项
	// keys: 要刷新的键列表
	// dstMap: 目标map的指针，用于接收结果，类型为*map[string]T
	// fallback: 批量回退函数
	// opts: 缓存选项，可以为nil使用默认选项
	// 返回: 错误信息
	MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error
}

// cacherImpl is the concrete implementation of the Cacher interface
type cacherImpl struct {
	store Store
}

// NewCacher creates a new Cacher instance with the given store
func NewCacher(store Store) Cacher {
	return &cacherImpl{store: store}
}

// Get implements the Get method of Cacher
func (c *cacherImpl) Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error) {
	// First try to get from cache
	found, err := c.store.Get(ctx, key, dst)
	if err != nil {
		return false, err
	}
	
	// If found in cache, return immediately
	if found {
		return true, nil
	}
	
	// Cache miss, execute fallback if provided
	if fallback != nil {
		value, found, err := fallback(ctx, key)
		if err != nil {
			return false, err
		}
		
		// If value found from fallback, cache it
		if found {
			// Use reflection to set the value to dst
			v := reflect.ValueOf(dst)
			if v.Kind() == reflect.Ptr && !v.IsNil() {
				v.Elem().Set(reflect.ValueOf(value))
			}
			
			// Set TTL if provided, otherwise use 0 (no expiration)
			ttl := time.Duration(0)
			if opts != nil {
				ttl = opts.TTL
			}
			
			// Cache the result
			items := map[string]interface{}{key: value}
			if err := c.store.MSet(ctx, items, ttl); err != nil {
				return false, err
			}
		}
		
		return found, nil
	}
	
	// No fallback provided and cache miss
	return false, nil
}

// MGet implements the MGet method of Cacher
func (c *cacherImpl) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	// First get all keys from cache
	resultMap := make(map[string]interface{})
	err := c.store.MGet(ctx, keys, &resultMap)
	if err != nil {
		return err
	}
	
	// Find missing keys
	missingKeys := []string{}
	for _, key := range keys {
		if _, exists := resultMap[key]; !exists {
			missingKeys = append(missingKeys, key)
		}
	}
	
	// If there are missing keys and a fallback is provided, execute it
	if len(missingKeys) > 0 && fallback != nil {
		fallbackResult, err := fallback(ctx, missingKeys)
		if err != nil {
			return err
		}
		
		// Add fallback results to the result map
		for k, v := range fallbackResult {
			resultMap[k] = v
		}
		
		// Cache the fallback results
		ttl := time.Duration(0)
		if opts != nil {
			ttl = opts.TTL
		}
		
		if err := c.store.MSet(ctx, fallbackResult, ttl); err != nil {
			return err
		}
	}
	
	// Use reflection to set the result map to dstMap
	v := reflect.ValueOf(dstMap)
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		mapVal := v.Elem()
		if mapVal.Kind() == reflect.Map {
			// Create a new map if it's nil
			if !mapVal.IsValid() || mapVal.IsNil() {
				mapType := mapVal.Type()
				mapVal = reflect.MakeMap(mapType)
				v.Elem().Set(mapVal)
			}
			
			// Copy results to the destination map
			for k, v := range resultMap {
				mapVal.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v))
			}
		}
	}
	
	return nil
}

// MDelete implements the MDelete method of Cacher
func (c *cacherImpl) MDelete(ctx context.Context, keys []string) (int64, error) {
	return c.store.Del(ctx, keys...)
}

// MRefresh implements the MRefresh method of Cacher
func (c *cacherImpl) MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	// Always execute fallback to get fresh data
	if fallback == nil {
		return nil
	}
	
	// Get fresh data from fallback
	freshData, err := fallback(ctx, keys)
	if err != nil {
		return err
	}
	
	// Cache the fresh data
	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}
	
	if err := c.store.MSet(ctx, freshData, ttl); err != nil {
		return err
	}
	
	// Use reflection to set the result to dstMap
	v := reflect.ValueOf(dstMap)
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		mapVal := v.Elem()
		if mapVal.Kind() == reflect.Map {
			// Create a new map if it's nil
			if !mapVal.IsValid() || mapVal.IsNil() {
				mapType := mapVal.Type()
				mapVal = reflect.MakeMap(mapType)
				v.Elem().Set(mapVal)
			}
			
			// Copy fresh data to the destination map
			for k, v := range freshData {
				mapVal.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v))
			}
		}
	}
	
	return nil
}