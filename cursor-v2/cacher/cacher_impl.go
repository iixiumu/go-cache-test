package cacher

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"go-cache/cacher/store"
)

// cacherImpl Cacher接口的实现
type cacherImpl struct {
	store store.Store
}

// NewCacher 创建新的Cacher实例
func NewCacher(store store.Store) Cacher {
	return &cacherImpl{store: store}
}

// Get 获取单个缓存项，缓存未命中时执行回退函数并缓存结果
func (c *cacherImpl) Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error) {
	// 首先尝试从缓存获取
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

	value, found, err := fallback(ctx, key)
	if err != nil {
		return false, err
	}

	if !found {
		return false, nil
	}

	// 将回退函数的结果设置到目标变量
	if err := setValue(dst, value); err != nil {
		return false, err
	}

	// 缓存结果
	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}

	err = c.store.MSet(ctx, map[string]interface{}{key: value}, ttl)
	if err != nil {
		// 缓存失败不影响返回值，只记录错误
		// 这里可以添加日志记录
	}

	return true, nil
}

// MGet 批量获取缓存项，支持部分命中和批量回退
func (c *cacherImpl) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	if len(keys) == 0 {
		return nil
	}

	// 首先尝试从缓存批量获取
	var cachedMap map[string]interface{}
	err := c.store.MGet(ctx, keys, &cachedMap)
	if err != nil {
		return err
	}

	// 找出未命中的键
	missedKeys := make([]string, 0)
	for _, key := range keys {
		if _, exists := cachedMap[key]; !exists {
			missedKeys = append(missedKeys, key)
		}
	}

	// 如果有未命中的键且提供了回退函数，则执行批量回退
	if len(missedKeys) > 0 && fallback != nil {
		fallbackResults, err := fallback(ctx, missedKeys)
		if err != nil {
			return err
		}

		// 将回退结果合并到缓存结果中
		for key, value := range fallbackResults {
			cachedMap[key] = value
		}

		// 缓存回退结果
		ttl := time.Duration(0)
		if opts != nil {
			ttl = opts.TTL
		}

		if len(fallbackResults) > 0 {
			err = c.store.MSet(ctx, fallbackResults, ttl)
			if err != nil {
				// 缓存失败不影响返回值，只记录错误
				// 这里可以添加日志记录
			}
		}
	}

	// 设置结果到目标map
	return setMapValue(dstMap, cachedMap)
}

// MDelete 批量清除缓存项
func (c *cacherImpl) MDelete(ctx context.Context, keys []string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	return c.store.Del(ctx, keys...)
}

// MRefresh 批量强制刷新缓存项
func (c *cacherImpl) MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	if len(keys) == 0 {
		return nil
	}

	// 强制删除现有缓存
	_, err := c.store.Del(ctx, keys...)
	if err != nil {
		return err
	}

	// 执行批量回退获取新数据
	if fallback == nil {
		return nil
	}

	fallbackResults, err := fallback(ctx, keys)
	if err != nil {
		return err
	}

	// 缓存新数据
	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}

	if len(fallbackResults) > 0 {
		err = c.store.MSet(ctx, fallbackResults, ttl)
		if err != nil {
			return err
		}
	}

	// 设置结果到目标map
	return setMapValue(dstMap, fallbackResults)
}

// setValue 使用反射将值设置到目标变量
func setValue(dst interface{}, value interface{}) error {
	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() != reflect.Ptr {
		return fmt.Errorf("dst must be a pointer")
	}

	srcValue := reflect.ValueOf(value)
	dstElem := dstValue.Elem()

	// 如果类型匹配，直接复制
	if srcValue.Type() == dstElem.Type() {
		dstElem.Set(srcValue)
		return nil
	}

	// 如果类型不匹配，尝试类型转换
	if srcValue.CanConvert(dstElem.Type()) {
		dstElem.Set(srcValue.Convert(dstElem.Type()))
		return nil
	}

	return fmt.Errorf("cannot convert %T to %T", value, dst)
}

// setMapValue 使用反射将map值设置到目标map
func setMapValue(dst interface{}, value map[string]interface{}) error {
	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() != reflect.Ptr || dstValue.Elem().Kind() != reflect.Map {
		return fmt.Errorf("dst must be a pointer to map")
	}

	dstMapValue := dstValue.Elem()
	dstMapValue.Set(reflect.ValueOf(value))

	return nil
}
