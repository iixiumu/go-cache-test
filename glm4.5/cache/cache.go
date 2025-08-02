package cache

import (
	"context"
	"errors"
	"reflect"

	"go-cache/internal"
	"go-cache/store"
)

// Cache Cacher接口的具体实现
type Cache struct {
	store store.Store
}

// NewCache 创建缓存实例
func NewCache(store store.Store) *Cache {
	return &Cache{store: store}
}

// Get 获取单个缓存项，缓存未命中时执行回退函数并缓存结果
func (c *Cache) Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error) {
	// 尝试从缓存获取
	found, err := c.store.Get(ctx, key, dst)
	if err != nil {
		return false, err
	}

	if found {
		return true, nil
	}

	// 缓存未命中，执行回退函数
	if fallback == nil {
		return false, nil
	}

	value, exists, err := fallback(ctx, key)
	if err != nil {
		return false, err
	}

	if !exists {
		return false, nil
	}

	// 将结果存入缓存
	ttl := internal.GetDefaultTTL(opts)
	items := map[string]interface{}{key: value}
	if err := c.store.MSet(ctx, items, ttl); err != nil {
		// 即使缓存失败也返回结果，但记录错误
		return true, errors.New("fallback succeeded but cache write failed")
	}

	// 将结果赋值给目标变量
	if err := internal.DeserializeValue([]byte(serializeValueOrDie(value)), dst); err != nil {
		return true, err
	}

	return true, nil
}

// MGet 批量获取缓存项，支持部分命中和批量回退
func (c *Cache) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	if len(keys) == 0 {
		return nil
	}

	// 如果没有提供回退函数，尝试从缓存获取
	if fallback == nil {
		return c.store.MGet(ctx, keys, dstMap)
	}

	// 对于有回退函数的情况，清空目标map以确保只包含最终结果
	dstMapValue := reflect.ValueOf(dstMap).Elem()
	dstMapValue.Set(reflect.MakeMap(dstMapValue.Type()))

	// 获取已缓存的值
	if err := c.store.MGet(ctx, keys, dstMap); err != nil {
		return err
	}

	// 找出未命中的键
	missingKeys := c.getMissingKeys(keys, dstMap)
	if len(missingKeys) == 0 {
		return nil
	}

	fallbackResults, err := fallback(ctx, missingKeys)
	if err != nil {
		return err
	}

	// 将回退结果存入缓存
	ttl := internal.GetDefaultTTL(opts)
	if len(fallbackResults) > 0 {
		if err := c.store.MSet(ctx, fallbackResults, ttl); err != nil {
			// 即使缓存失败也继续处理结果
		}
	}

	// 将回退结果合并到目标map中
	for key, value := range fallbackResults {
		if err := internal.SetMapValue(dstMap, key, value); err != nil {
			return err
		}
	}

	return nil
}

// MDelete 批量清除缓存项
func (c *Cache) MDelete(ctx context.Context, keys []string) (int64, error) {
	return c.store.Del(ctx, keys...)
}

// MRefresh 批量强制刷新缓存项
func (c *Cache) MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	if len(keys) == 0 {
		return nil
	}

	// 先删除现有缓存
	if _, err := c.store.Del(ctx, keys...); err != nil {
		return err
	}

	// 执行回退函数获取最新数据
	if fallback == nil {
		return nil
	}

	fallbackResults, err := fallback(ctx, keys)
	if err != nil {
		return err
	}

	// 将最新数据存入缓存
	ttl := internal.GetDefaultTTL(opts)
	if len(fallbackResults) > 0 {
		if err := c.store.MSet(ctx, fallbackResults, ttl); err != nil {
			return err
		}
	}

	// 将结果赋值给目标map
	for key, value := range fallbackResults {
		if err := internal.SetMapValue(dstMap, key, value); err != nil {
			return err
		}
	}

	return nil
}

// getMissingKeys 获取未命中的键
func (c *Cache) getMissingKeys(keys []string, dstMap interface{}) []string {
	keyType, _, err := internal.GetTypeOfMap(dstMap)
	if err != nil {
		return nil
	}

	dstMapValue := reflect.ValueOf(dstMap).Elem()
	var missingKeys []string

	for _, key := range keys {
		keyValue := reflect.ValueOf(key)
		if keyValue.Type().ConvertibleTo(keyType) {
			keyValue = keyValue.Convert(keyType)
			if !dstMapValue.MapIndex(keyValue).IsValid() {
				missingKeys = append(missingKeys, key)
			}
		} else {
			missingKeys = append(missingKeys, key)
		}
	}

	return missingKeys
}

// serializeValueOrDie 序列化值（忽略错误）
func serializeValueOrDie(value interface{}) []byte {
	data, _ := internal.SerializeValue(value)
	return data
}

// GetStore 获取底层存储
func (c *Cache) GetStore() store.Store {
	return c.store
}

// Close 关闭缓存
func (c *Cache) Close() error {
	if closer, ok := c.store.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}