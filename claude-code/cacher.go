package cache

import (
	"context"
	"fmt"
	"reflect"
	"time"
)

// DefaultCacher 默认缓存实现，提供带回退机制的缓存操作
type DefaultCacher struct {
	store      Store
	defaultTTL time.Duration
}

// NewCacher 创建新的缓存实例
func NewCacher(store Store) *DefaultCacher {
	return &DefaultCacher{
		store:      store,
		defaultTTL: time.Hour, // 默认1小时TTL
	}
}

// NewCacherWithTTL 创建带默认TTL的缓存实例
func NewCacherWithTTL(store Store, defaultTTL time.Duration) *DefaultCacher {
	return &DefaultCacher{
		store:      store,
		defaultTTL: defaultTTL,
	}
}

// Get 获取单个缓存项，缓存未命中时执行回退函数并缓存结果
func (c *DefaultCacher) Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error) {
	// 尝试从缓存获取
	found, err := c.store.Get(ctx, key, dst)
	if err != nil {
		return false, fmt.Errorf("store get failed: %w", err)
	}

	if found {
		return true, nil
	}

	// 缓存未命中，调用回退函数
	if fallback == nil {
		return false, nil
	}

	value, found, err := fallback(ctx, key)
	if err != nil {
		return false, fmt.Errorf("fallback function failed: %w", err)
	}

	if !found {
		return false, nil
	}

	// 缓存回退函数的结果
	ttl := c.defaultTTL
	if opts != nil && opts.TTL > 0 {
		ttl = opts.TTL
	}

	items := map[string]interface{}{key: value}
	if err := c.store.MSet(ctx, items, ttl); err != nil {
		// 缓存设置失败不影响返回结果，只记录错误
		// 在生产环境中可能需要更好的错误处理策略
	}

	// 将值设置到目标变量
	serialized, err := SerializeValue(value)
	if err != nil {
		return false, fmt.Errorf("serialize fallback value failed: %w", err)
	}

	if err := DeserializeValue(serialized, dst); err != nil {
		return false, fmt.Errorf("deserialize fallback value failed: %w", err)
	}

	return true, nil
}

// MGet 批量获取缓存项，支持部分命中和批量回退
func (c *DefaultCacher) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	if len(keys) == 0 {
		return nil
	}

	if err := ValidateMapPointer(dstMap); err != nil {
		return err
	}

	// 从缓存批量获取
	if err := c.store.MGet(ctx, keys, dstMap); err != nil {
		return fmt.Errorf("store mget failed: %w", err)
	}

	// 如果没有回退函数，直接返回
	if fallback == nil {
		return nil
	}

	// 检查哪些键未命中
	mapValue := reflect.ValueOf(dstMap).Elem()
	var missedKeys []string
	
	for _, key := range keys {
		if !mapValue.MapIndex(reflect.ValueOf(key)).IsValid() {
			missedKeys = append(missedKeys, key)
		}
	}

	if len(missedKeys) == 0 {
		return nil // 所有键都命中了
	}

	// 调用批量回退函数
	fallbackData, err := fallback(ctx, missedKeys)
	if err != nil {
		return fmt.Errorf("batch fallback function failed: %w", err)
	}

	if len(fallbackData) == 0 {
		return nil
	}

	// 缓存回退函数的结果
	ttl := c.defaultTTL
	if opts != nil && opts.TTL > 0 {
		ttl = opts.TTL
	}

	if err := c.store.MSet(ctx, fallbackData, ttl); err != nil {
		// 缓存设置失败不影响返回结果
	}

	// 将回退数据设置到目标map中
	for key, value := range fallbackData {
		serialized, err := SerializeValue(value)
		if err != nil {
			return fmt.Errorf("serialize fallback value failed for key %s: %w", key, err)
		}

		if err := SetMapValue(dstMap, key, serialized); err != nil {
			return fmt.Errorf("set fallback value failed for key %s: %w", key, err)
		}
	}

	return nil
}

// MDelete 批量清除缓存项
func (c *DefaultCacher) MDelete(ctx context.Context, keys []string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	deleted, err := c.store.Del(ctx, keys...)
	if err != nil {
		return 0, fmt.Errorf("store delete failed: %w", err)
	}

	return deleted, nil
}

// MRefresh 批量强制刷新缓存项
func (c *DefaultCacher) MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	if len(keys) == 0 {
		return nil
	}

	if err := ValidateMapPointer(dstMap); err != nil {
		return err
	}

	if fallback == nil {
		return fmt.Errorf("fallback function is required for refresh")
	}

	// 先删除现有缓存
	if _, err := c.store.Del(ctx, keys...); err != nil {
		return fmt.Errorf("delete existing cache failed: %w", err)
	}

	// 调用回退函数获取新数据
	fallbackData, err := fallback(ctx, keys)
	if err != nil {
		return fmt.Errorf("refresh fallback function failed: %w", err)
	}

	// 缓存新数据
	ttl := c.defaultTTL
	if opts != nil && opts.TTL > 0 {
		ttl = opts.TTL
	}

	if len(fallbackData) > 0 {
		if err := c.store.MSet(ctx, fallbackData, ttl); err != nil {
			return fmt.Errorf("cache refresh data failed: %w", err)
		}
	}

	// 将数据设置到目标map中
	for key, value := range fallbackData {
		serialized, err := SerializeValue(value)
		if err != nil {
			return fmt.Errorf("serialize refresh value failed for key %s: %w", key, err)
		}

		if err := SetMapValue(dstMap, key, serialized); err != nil {
			return fmt.Errorf("set refresh value failed for key %s: %w", key, err)
		}
	}

	return nil
}