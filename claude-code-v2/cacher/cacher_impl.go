package cacher

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"go-cache/cacher/store"
)

// CacherImpl Cacher接口的实现
type CacherImpl struct {
	store store.Store
}

// NewCacher 创建新的Cacher实例
func NewCacher(store store.Store) Cacher {
	return &CacherImpl{
		store: store,
	}
}

// Get 获取单个缓存项，缓存未命中时执行回退函数并缓存结果
func (c *CacherImpl) Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error) {
	// 首先尝试从缓存获取
	found, err := c.store.Get(ctx, key, dst)
	if err != nil {
		return false, fmt.Errorf("failed to get from store: %w", err)
	}

	// 如果缓存命中，直接返回
	if found {
		return true, nil
	}

	// 缓存未命中，调用fallback函数
	if fallback == nil {
		return false, nil
	}

	value, found, err := fallback(ctx, key)
	if err != nil {
		return false, fmt.Errorf("fallback error: %w", err)
	}

	if !found {
		return false, nil
	}

	// 将fallback的结果复制到dst
	if err := c.copyValue(value, dst); err != nil {
		return false, fmt.Errorf("failed to copy fallback value: %w", err)
	}

	// 缓存fallback的结果
	ttl := c.getTTL(opts)
	items := map[string]interface{}{key: value}
	if err := c.store.MSet(ctx, items, ttl); err != nil {
		// 记录错误但不影响返回结果
		// 在实际生产环境中，这里应该使用日志系统
		_ = fmt.Errorf("failed to cache value: %w", err)
	}

	return true, nil
}

// MGet 批量获取缓存项，支持部分命中和批量回退
func (c *CacherImpl) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	if len(keys) == 0 {
		return nil
	}

	// 验证dstMap是map指针
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr || dstMapValue.Elem().Kind() != reflect.Map {
		return fmt.Errorf("dstMap must be a pointer to map")
	}

	mapValue := dstMapValue.Elem()
	mapType := mapValue.Type()

	// 如果map为nil，初始化它
	if mapValue.IsNil() {
		mapValue.Set(reflect.MakeMap(mapType))
	}

	// 首先尝试从缓存批量获取
	err := c.store.MGet(ctx, keys, dstMap)
	if err != nil {
		return fmt.Errorf("failed to mget from store: %w", err)
	}

	// 检查哪些键未命中缓存
	currentMap := mapValue.Interface()
	missedKeys := make([]string, 0)
	
	currentMapValue := reflect.ValueOf(currentMap)
	for _, key := range keys {
		keyValue := reflect.ValueOf(key)
		if !currentMapValue.MapIndex(keyValue).IsValid() {
			missedKeys = append(missedKeys, key)
		}
	}

	// 如果所有键都命中缓存，直接返回
	if len(missedKeys) == 0 {
		return nil
	}

	// 如果有未命中的键且有fallback函数，调用fallback
	if fallback != nil {
		fallbackResults, err := fallback(ctx, missedKeys)
		if err != nil {
			return fmt.Errorf("batch fallback error: %w", err)
		}

		// 处理fallback结果
		valueType := mapType.Elem()
		for key, value := range fallbackResults {
			// 创建值类型的新实例
			valuePtr := reflect.New(valueType)
			
			// 复制值
			if err := c.copyValue(value, valuePtr.Interface()); err != nil {
				return fmt.Errorf("failed to copy fallback value for key %s: %w", key, err)
			}

			// 设置到map中
			keyValue := reflect.ValueOf(key)
			mapValue.SetMapIndex(keyValue, valuePtr.Elem())
		}

		// 缓存fallback的结果
		ttl := c.getTTL(opts)
		if err := c.store.MSet(ctx, fallbackResults, ttl); err != nil {
			// 记录错误但不影响返回结果
			_ = fmt.Errorf("failed to cache fallback values: %w", err)
		}
	}

	return nil
}

// MDelete 批量清除缓存项
func (c *CacherImpl) MDelete(ctx context.Context, keys []string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	deletedCount, err := c.store.Del(ctx, keys...)
	if err != nil {
		return 0, fmt.Errorf("failed to delete from store: %w", err)
	}

	return deletedCount, nil
}

// MRefresh 批量强制刷新缓存项
func (c *CacherImpl) MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	if len(keys) == 0 {
		return nil
	}

	if fallback == nil {
		return fmt.Errorf("fallback function is required for refresh")
	}

	// 验证dstMap是map指针
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr || dstMapValue.Elem().Kind() != reflect.Map {
		return fmt.Errorf("dstMap must be a pointer to map")
	}

	mapValue := dstMapValue.Elem()
	mapType := mapValue.Type()
	valueType := mapType.Elem()

	// 如果map为nil，初始化它
	if mapValue.IsNil() {
		mapValue.Set(reflect.MakeMap(mapType))
	}

	// 调用fallback获取最新数据
	fallbackResults, err := fallback(ctx, keys)
	if err != nil {
		return fmt.Errorf("batch fallback error for refresh: %w", err)
	}

	// 处理fallback结果
	for key, value := range fallbackResults {
		// 创建值类型的新实例
		valuePtr := reflect.New(valueType)
		
		// 复制值
		if err := c.copyValue(value, valuePtr.Interface()); err != nil {
			return fmt.Errorf("failed to copy refresh value for key %s: %w", key, err)
		}

		// 设置到map中
		keyValue := reflect.ValueOf(key)
		mapValue.SetMapIndex(keyValue, valuePtr.Elem())
	}

	// 更新缓存
	ttl := c.getTTL(opts)
	if err := c.store.MSet(ctx, fallbackResults, ttl); err != nil {
		return fmt.Errorf("failed to refresh cache: %w", err)
	}

	return nil
}

// copyValue 复制值，处理不同类型的复制逻辑
func (c *CacherImpl) copyValue(src, dst interface{}) error {
	srcValue := reflect.ValueOf(src)
	dstValue := reflect.ValueOf(dst)

	// dst必须是指针
	if dstValue.Kind() != reflect.Ptr {
		return fmt.Errorf("dst must be a pointer")
	}

	dstElem := dstValue.Elem()
	
	// 如果类型相同，直接设置
	if srcValue.Type().AssignableTo(dstElem.Type()) {
		dstElem.Set(srcValue)
		return nil
	}

	// 如果类型不同但可以转换
	if srcValue.Type().ConvertibleTo(dstElem.Type()) {
		dstElem.Set(srcValue.Convert(dstElem.Type()))
		return nil
	}

	return fmt.Errorf("cannot copy value of type %T to %T", src, dst)
}

// getTTL 从选项中获取TTL，如果选项为nil则返回0
func (c *CacherImpl) getTTL(opts *CacheOptions) time.Duration {
	if opts == nil {
		return 0
	}
	return opts.TTL
}

// 确保CacherImpl实现了Cacher接口
var _ Cacher = (*CacherImpl)(nil)
