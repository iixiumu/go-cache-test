package cacher

import (
	"context"
	"go-cache/store"
	"reflect"
)

// cacher 是 Cacher 接口的实现
type cacher struct {
	store store.Store
}

// NewCacher 创建一个新的 Cacher 实例
func NewCacher(store store.Store) Cacher {
	return &cacher{
		store: store,
	}
}

// Get 获取单个缓存项，缓存未命中时执行回退函数并缓存结果
func (c *cacher) Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error) {
	// 先尝试从缓存中获取
	found, err := c.store.Get(ctx, key, dst)
	if err != nil {
		return false, err
	}

	// 如果找到了，直接返回
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

	// 如果回退函数没有返回值，则返回未找到
	if !found {
		return false, nil
	}

	// 将回退获取的值存入缓存
	if opts == nil {
		opts = &CacheOptions{TTL: 0}
	}

	// 使用反射创建一个临时map来存储值
	tempMap := map[string]interface{}{
		key: value,
	}
	err = c.store.MSet(ctx, tempMap, opts.TTL)
	if err != nil {
		return false, err
	}

	// 将值设置到dst中
	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() == reflect.Ptr && !dstValue.IsNil() {
		dstElem := dstValue.Elem()
		valueReflect := reflect.ValueOf(value)
		if valueReflect.Type().AssignableTo(dstElem.Type()) {
			dstElem.Set(valueReflect)
		}
	}

	return true, nil
}

// MGet 批量获取缓存项，支持部分命中和批量回退
func (c *cacher) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	// 先尝试从缓存中批量获取
	err := c.store.MGet(ctx, keys, dstMap)
	if err != nil {
		return err
	}

	// 检查哪些键在缓存中命中了
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr || dstMapValue.IsNil() {
		return nil
	}

	// 获取实际的map
	mapValue := dstMapValue.Elem()

	// 找出未命中的键
	missedKeys := make([]string, 0)

	// 通过反射遍历map来确定哪些键未命中
	if mapValue.Kind() == reflect.Map {
		for _, key := range keys {
			if mapValue.MapIndex(reflect.ValueOf(key)).Kind() == reflect.Invalid {
				missedKeys = append(missedKeys, key)
			}
		}
	}

	// 如果所有键都命中了，直接返回
	if len(missedKeys) == 0 {
		return nil
	}

	// 如果有未命中的键，执行批量回退函数
	if fallback == nil {
		return nil
	}

	// 执行回退函数
	fallbackResult, err := fallback(ctx, missedKeys)
	if err != nil {
		return err
	}

	// 将回退结果存入缓存
	if len(fallbackResult) > 0 {
		if opts == nil {
			opts = &CacheOptions{TTL: 0}
		}

		err = c.store.MSet(ctx, fallbackResult, opts.TTL)
		if err != nil {
			return err
		}

		// 更新返回结果
		for key, value := range fallbackResult {
			mapValue.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
		}
	}

	return nil
}

// MDelete 批量清除缓存项
func (c *cacher) MDelete(ctx context.Context, keys []string) (int64, error) {
	return c.store.Del(ctx, keys...)
}

// MRefresh 批量强制刷新缓存项
func (c *cacher) MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	// 先删除旧缓存
	_, err := c.store.Del(ctx, keys...)
	if err != nil {
		return err
	}

	// 然后重新获取
	return c.MGet(ctx, keys, dstMap, fallback, opts)
}
