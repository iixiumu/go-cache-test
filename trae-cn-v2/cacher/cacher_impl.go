package cacher

import (
	"context"
	"errors"
	"reflect"

	"go-cache/cacher/store"
)

// cacherImpl 实现了Cacher接口
type cacherImpl struct {
	store store.Store
}

// NewCacher 创建一个新的Cacher实例
// store: 底层存储后端
func NewCacher(store store.Store) Cacher {
	return &cacherImpl{
		store: store,
	}
}

// Get 获取单个缓存项，缓存未命中时执行回退函数并缓存结果
func (c *cacherImpl) Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error) {
	// 默认选项
	if opts == nil {
		opts = &CacheOptions{}
	}

	// 尝试从缓存获取
	found, err := c.store.Get(ctx, key, dst)
	if err != nil {
		return false, err
	}

	// 缓存命中
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

	// 回退函数未找到值
	if !found {
		return false, nil
	}

	// 缓存回退函数的结果
	if err := c.store.MSet(ctx, map[string]interface{}{key: value}, opts.TTL); err != nil {
		return false, err
	}

	// 设置结果到dst
	dstVal := reflect.ValueOf(dst)
	if dstVal.Kind() != reflect.Ptr {
		return false, errors.New("dst must be a pointer")
	}

	dstElem := dstVal.Elem()
	valVal := reflect.ValueOf(value)
	if !valVal.Type().AssignableTo(dstElem.Type()) {
		return false, errors.New("type mismatch")
	}

	dstElem.Set(valVal)

	return true, nil
}

// MGet 批量获取缓存项，支持部分命中和批量回退
func (c *cacherImpl) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	// 默认选项
	if opts == nil {
		opts = &CacheOptions{}
	}

	// 尝试从缓存获取
	if err := c.store.MGet(ctx, keys, dstMap); err != nil {
		return err
	}

	// 检查哪些键未命中
	mapVal := reflect.ValueOf(dstMap)
	if mapVal.Kind() != reflect.Ptr || mapVal.Elem().Kind() != reflect.Map {
		return errors.New("dstMap must be a pointer to a map")
	}

	var missingKeys []string
	for _, key := range keys {
		if !mapVal.Elem().MapIndex(reflect.ValueOf(key)).IsValid() {
			missingKeys = append(missingKeys, key)
		}
	}

	// 所有键都命中
	if len(missingKeys) == 0 || fallback == nil {
		return nil
	}

	// 回退获取未命中的键
	fallbackResult, err := fallback(ctx, missingKeys)
	if err != nil {
		return err
	}

	// 缓存回退结果
	if len(fallbackResult) > 0 {
		if err := c.store.MSet(ctx, fallbackResult, opts.TTL); err != nil {
			return err
		}

		// 更新结果map
		for key, value := range fallbackResult {
			valVal := reflect.ValueOf(value)
			if valVal.Type().AssignableTo(mapVal.Elem().Type().Elem()) {
				mapVal.Elem().SetMapIndex(reflect.ValueOf(key), valVal)
			}
		}
	}

	return nil
}

// MDelete 批量清除缓存项
func (c *cacherImpl) MDelete(ctx context.Context, keys []string) (int64, error) {
	return c.store.Del(ctx, keys...)
}

// MRefresh 批量强制刷新缓存项
func (c *cacherImpl) MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	// 删除现有缓存项
	_, err := c.store.Del(ctx, keys...)
	if err != nil {
		return err
	}

	// 调用MGet获取新值
	return c.MGet(ctx, keys, dstMap, fallback, opts)
}