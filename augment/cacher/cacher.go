package cacher

import (
	"context"
	"fmt"
	"reflect"
	"time"

	cache "go-cache"
)

// DefaultCacher 默认缓存实现
type DefaultCacher struct {
	store      cache.Store
	defaultTTL time.Duration
}

// NewDefaultCacher 创建新的默认缓存实例
func NewDefaultCacher(store cache.Store, defaultTTL time.Duration) *DefaultCacher {
	return &DefaultCacher{
		store:      store,
		defaultTTL: defaultTTL,
	}
}

// Get 获取单个缓存项，缓存未命中时执行回退函数并缓存结果
func (c *DefaultCacher) Get(ctx context.Context, key string, dst interface{}, fallback cache.FallbackFunc, opts *cache.CacheOptions) (bool, error) {
	// 验证dst是指针
	if reflect.TypeOf(dst).Kind() != reflect.Ptr {
		return false, fmt.Errorf("dst must be a pointer")
	}

	// 首先尝试从缓存获取
	found, err := c.store.Get(ctx, key, dst)
	if err != nil {
		return false, fmt.Errorf("failed to get from store: %w", err)
	}

	if found {
		return true, nil
	}

	// 缓存未命中，执行回退函数
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

	// 将回退函数的结果赋值给dst
	if err := assignValue(dst, value); err != nil {
		return false, fmt.Errorf("failed to assign fallback value: %w", err)
	}

	// 缓存回退函数的结果
	ttl := c.defaultTTL
	if opts != nil && opts.TTL > 0 {
		ttl = opts.TTL
	}

	items := map[string]interface{}{key: value}
	if err := c.store.MSet(ctx, items, ttl); err != nil {
		// 缓存失败不影响返回结果，只记录错误
		// 在实际应用中可能需要日志记录
	}

	return true, nil
}

// MGet 批量获取缓存项，支持部分命中和批量回退
func (c *DefaultCacher) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback cache.BatchFallbackFunc, opts *cache.CacheOptions) error {
	// 验证dstMap是map指针
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr || dstMapValue.Elem().Kind() != reflect.Map {
		return fmt.Errorf("dstMap must be a pointer to map")
	}

	// 从缓存批量获取
	if err := c.store.MGet(ctx, keys, dstMap); err != nil {
		return fmt.Errorf("failed to mget from store: %w", err)
	}

	// 检查哪些键未命中
	mapValue := dstMapValue.Elem()
	var missedKeys []string
	for _, key := range keys {
		keyValue := reflect.ValueOf(key)
		if !mapValue.MapIndex(keyValue).IsValid() {
			missedKeys = append(missedKeys, key)
		}
	}

	// 如果所有键都命中或没有回退函数，直接返回
	if len(missedKeys) == 0 || fallback == nil {
		return nil
	}

	// 执行批量回退函数
	fallbackData, err := fallback(ctx, missedKeys)
	if err != nil {
		return fmt.Errorf("batch fallback function failed: %w", err)
	}

	// 将回退数据添加到结果map中
	mapType := mapValue.Type()
	valueType := mapType.Elem()
	
	for key, value := range fallbackData {
		convertedValue, err := convertValue(value, valueType)
		if err != nil {
			return fmt.Errorf("failed to convert fallback value for key %s: %w", key, err)
		}
		mapValue.SetMapIndex(reflect.ValueOf(key), convertedValue)
	}

	// 缓存回退函数的结果
	if len(fallbackData) > 0 {
		ttl := c.defaultTTL
		if opts != nil && opts.TTL > 0 {
			ttl = opts.TTL
		}

		if err := c.store.MSet(ctx, fallbackData, ttl); err != nil {
			// 缓存失败不影响返回结果，只记录错误
			// 在实际应用中可能需要日志记录
		}
	}

	return nil
}

// MDelete 批量清除缓存项
func (c *DefaultCacher) MDelete(ctx context.Context, keys []string) (int64, error) {
	return c.store.Del(ctx, keys...)
}

// MRefresh 批量强制刷新缓存项
func (c *DefaultCacher) MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback cache.BatchFallbackFunc, opts *cache.CacheOptions) error {
	// 先删除现有缓存
	_, err := c.store.Del(ctx, keys...)
	if err != nil {
		return fmt.Errorf("failed to delete keys for refresh: %w", err)
	}

	// 验证dstMap是map指针
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr || dstMapValue.Elem().Kind() != reflect.Map {
		return fmt.Errorf("dstMap must be a pointer to map")
	}

	// 清空目标map
	mapValue := dstMapValue.Elem()
	mapValue.Set(reflect.MakeMap(mapValue.Type()))

	// 如果没有回退函数，直接返回
	if fallback == nil {
		return nil
	}

	// 执行批量回退函数获取新数据
	fallbackData, err := fallback(ctx, keys)
	if err != nil {
		return fmt.Errorf("batch fallback function failed during refresh: %w", err)
	}

	// 将新数据添加到结果map中
	mapType := mapValue.Type()
	valueType := mapType.Elem()
	
	for key, value := range fallbackData {
		convertedValue, err := convertValue(value, valueType)
		if err != nil {
			return fmt.Errorf("failed to convert fallback value for key %s during refresh: %w", key, err)
		}
		mapValue.SetMapIndex(reflect.ValueOf(key), convertedValue)
	}

	// 缓存新数据
	if len(fallbackData) > 0 {
		ttl := c.defaultTTL
		if opts != nil && opts.TTL > 0 {
			ttl = opts.TTL
		}

		if err := c.store.MSet(ctx, fallbackData, ttl); err != nil {
			return fmt.Errorf("failed to cache refreshed data: %w", err)
		}
	}

	return nil
}

// assignValue 将value赋值给dst指针指向的变量
func assignValue(dst interface{}, value interface{}) error {
	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() != reflect.Ptr {
		return fmt.Errorf("dst must be a pointer")
	}

	dstElem := dstValue.Elem()
	valueReflect := reflect.ValueOf(value)

	if !valueReflect.Type().AssignableTo(dstElem.Type()) {
		// 尝试类型转换
		if valueReflect.Type().ConvertibleTo(dstElem.Type()) {
			valueReflect = valueReflect.Convert(dstElem.Type())
		} else {
			return fmt.Errorf("cannot assign %T to %T", value, dst)
		}
	}

	dstElem.Set(valueReflect)
	return nil
}

// convertValue 将value转换为targetType类型
func convertValue(value interface{}, targetType reflect.Type) (reflect.Value, error) {
	valueReflect := reflect.ValueOf(value)
	
	if valueReflect.Type().AssignableTo(targetType) {
		return valueReflect, nil
	}
	
	if valueReflect.Type().ConvertibleTo(targetType) {
		return valueReflect.Convert(targetType), nil
	}
	
	return reflect.Value{}, fmt.Errorf("cannot convert %T to %s", value, targetType)
}
