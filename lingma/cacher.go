package cache

import (
	"context"
	"reflect"
	"time"
)

// cacher 实现了Cacher接口
type cacher struct {
	store Store
}

// NewCacher 创建一个新的Cacher实例
func NewCacher(store Store) Cacher {
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

	// 如果没有找到且没有提供回退函数，返回未找到
	if fallback == nil {
		return false, nil
	}

	// 执行回退函数获取数据
	value, found, err := fallback(ctx, key)
	if err != nil || !found {
		return found, err
	}

	// 将回退函数获取到的数据存入缓存
	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}

	err = c.store.MSet(ctx, map[string]interface{}{key: value}, ttl)
	if err != nil {
		// 即使缓存失败，也返回回退函数获取到的值
		return true, nil
	}

	// 将值设置到dst中
	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() != reflect.Ptr || dstValue.IsNil() {
		return true, nil
	}

	reflect.Indirect(dstValue).Set(reflect.ValueOf(value))
	return true, nil
}

// MGet 批量获取缓存项，支持部分命中和批量回退
func (c *cacher) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	// 先尝试从缓存中批量获取
	err := c.store.MGet(ctx, keys, dstMap)
	if err != nil {
		return err
	}

	// 检查哪些键未命中
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr || dstMapValue.IsNil() {
		return nil
	}

	dstMapElem := dstMapValue.Elem()
	hitKeys := make(map[string]bool)
	dstMapIter := dstMapElem.MapRange()
	for dstMapIter.Next() {
		hitKeys[dstMapIter.Key().String()] = true
	}

	// 找出未命中的键
	var missingKeys []string
	for _, key := range keys {
		if !hitKeys[key] {
			missingKeys = append(missingKeys, key)
		}
	}

	// 如果没有未命中的键，直接返回
	if len(missingKeys) == 0 {
		return nil
	}

	// 如果没有提供回退函数，直接返回
	if fallback == nil {
		return nil
	}

	// 执行回退函数获取未命中的数据
	fallbackValues, err := fallback(ctx, missingKeys)
	if err != nil {
		return err
	}

	// 将回退函数获取到的数据存入缓存
	if len(fallbackValues) > 0 {
		ttl := time.Duration(0)
		if opts != nil {
			ttl = opts.TTL
		}

		err = c.store.MSet(ctx, fallbackValues, ttl)
		if err != nil {
			// 即使缓存失败，也继续处理
		}

		// 将回退函数获取到的数据合并到结果中
		for k, v := range fallbackValues {
			dstMapElem.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v))
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
	// 不检查缓存，直接执行回退函数
	if fallback == nil {
		return nil
	}

	values, err := fallback(ctx, keys)
	if err != nil {
		return err
	}

	// 更新缓存
	if len(values) > 0 {
		ttl := time.Duration(0)
		if opts != nil {
			ttl = opts.TTL
		}

		err = c.store.MSet(ctx, values, ttl)
		if err != nil {
			// 即使缓存失败，也继续处理
		}

		// 将数据合并到结果中
		dstMapValue := reflect.ValueOf(dstMap)
		if dstMapValue.Kind() != reflect.Ptr || dstMapValue.IsNil() {
			return nil
		}

		dstMapElem := dstMapValue.Elem()
		for k, v := range values {
			dstMapElem.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v))
		}
	}

	return nil
}