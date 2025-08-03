package cacher

import (
	"context"
	"fmt"
	"reflect"

	"go-cache/cacher/store"
)

// CacherImpl Cacher接口实现
type CacherImpl struct {
	store store.Store
}

// NewCacher 创建新的Cacher实例
func NewCacher(store store.Store) *CacherImpl {
	return &CacherImpl{
		store: store,
	}
}

// Get 获取单个缓存项，缓存未命中时执行回退函数并缓存结果
func (c *CacherImpl) Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error) {
	if opts == nil {
		opts = &CacheOptions{TTL: 0}
	}

	// 先从缓存获取
	found, err := c.store.Get(ctx, key, dst)
	if err != nil {
		return false, fmt.Errorf("cache get failed: %w", err)
	}
	if found {
		return true, nil
	}

	// 缓存未命中，执行回退函数
	value, found, err := fallback(ctx, key)
	if err != nil {
		return false, fmt.Errorf("fallback function failed: %w", err)
	}
	if !found {
		return false, nil
	}

	// 缓存结果
	items := map[string]interface{}{
		key: value,
	}
	if err := c.store.MSet(ctx, items, opts.TTL); err != nil {
		// 缓存失败不返回错误，只记录日志
		// 在实际应用中，这里可以添加日志记录
	}

	// 设置返回值
	if err := setValue(dst, value); err != nil {
		return false, fmt.Errorf("failed to set result value: %w", err)
	}

	return true, nil
}

// MGet 批量获取缓存项，支持部分命中和批量回退
func (c *CacherImpl) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	if opts == nil {
		opts = &CacheOptions{TTL: 0}
	}

	if len(keys) == 0 {
		return nil
	}

	// 从缓存获取已存在的值
	cacheResults := make(map[string]interface{})
	if err := c.store.MGet(ctx, keys, &cacheResults); err != nil {
		return fmt.Errorf("cache mget failed: %w", err)
	}

	// 找出未命中的键
	missedKeys := make([]string, 0)
	for _, key := range keys {
		if _, exists := cacheResults[key]; !exists {
			missedKeys = append(missedKeys, key)
		}
	}

	// 如果有未命中的键，执行批量回退
	if len(missedKeys) > 0 {
		fallbackResults, err := fallback(ctx, missedKeys)
		if err != nil {
			return fmt.Errorf("batch fallback failed: %w", err)
		}

		// 缓存回退结果
		if len(fallbackResults) > 0 {
			if err := c.store.MSet(ctx, fallbackResults, opts.TTL); err != nil {
				// 缓存失败不返回错误
			}
		}

		// 合并结果
		for key, value := range fallbackResults {
			cacheResults[key] = value
		}
	}

	// 设置返回值
	return setMapValue(dstMap, cacheResults, keys)
}

// MDelete 批量清除缓存项
func (c *CacherImpl) MDelete(ctx context.Context, keys []string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	deleted, err := c.store.Del(ctx, keys...)
	if err != nil {
		return 0, fmt.Errorf("cache delete failed: %w", err)
	}

	return deleted, nil
}

// MRefresh 批量强制刷新缓存项
func (c *CacherImpl) MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	if opts == nil {
		opts = &CacheOptions{TTL: 0}
	}

	if len(keys) == 0 {
		return nil
	}

	// 删除旧的缓存项
	if _, err := c.store.Del(ctx, keys...); err != nil {
		return fmt.Errorf("cache delete failed: %w", err)
	}

	// 执行批量回退获取新值
	fallbackResults, err := fallback(ctx, keys)
	if err != nil {
		return fmt.Errorf("batch fallback failed: %w", err)
	}

	// 缓存新结果
	if len(fallbackResults) > 0 {
		if err := c.store.MSet(ctx, fallbackResults, opts.TTL); err != nil {
			// 缓存失败不返回错误
		}
	}

	// 设置返回值
	return setMapValue(dstMap, fallbackResults, keys)
}

// setValue 使用反射设置值
func setValue(dst interface{}, value interface{}) error {
	if dst == nil {
		return fmt.Errorf("destination is nil")
	}

	ptr := reflect.ValueOf(dst)
	if ptr.Kind() != reflect.Ptr {
		return fmt.Errorf("destination must be a pointer")
	}

	val := ptr.Elem()
	if !val.CanSet() {
		return fmt.Errorf("cannot set destination value")
	}

	// 设置值
	val.Set(reflect.ValueOf(value))
	return nil
}

// setMapValue 使用反射设置map值，保持原始键的顺序
func setMapValue(dstMap interface{}, values map[string]interface{}, keys []string) error {
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

	// 清空目标map
	mapVal.Set(reflect.MakeMap(mapVal.Type()))

	// 按原始键的顺序设置值
	for _, key := range keys {
		if value, exists := values[key]; exists {
			mapVal.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
		}
	}

	return nil
}