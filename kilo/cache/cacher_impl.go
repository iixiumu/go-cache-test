package cache

import (
	"context"
	"reflect"
	"time"
)

// cacherImpl 实现了Cacher接口
type cacherImpl struct {
	store Store
}

// NewCacher 创建一个新的Cacher实例
func NewCacher(store Store) Cacher {
	return &cacherImpl{
		store: store,
	}
}

// Get 获取单个缓存项，缓存未命中时执行回退函数并缓存结果
func (c *cacherImpl) Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error) {
	// 先尝试从缓存获取
	found, err := c.store.Get(ctx, key, dst)
	if err != nil {
		return false, err
	}

	// 如果缓存命中，直接返回
	if found {
		return true, nil
	}

	// 如果没有提供回退函数，返回未找到
	if fallback == nil {
		return false, nil
	}

	// 执行回退函数获取数据
	value, found, err := fallback(ctx, key)
	if err != nil {
		return false, err
	}

	// 如果回退函数没有找到数据，返回未找到
	if !found {
		return false, nil
	}

	// 将回退函数获取的数据存入缓存
	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}

	err = c.store.MSet(ctx, map[string]interface{}{key: value}, ttl)
	if err != nil {
		return false, err
	}

	// 将值赋给dst
	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() != reflect.Ptr {
		return false, nil
	}

	dstElem := dstValue.Elem()
	if !dstElem.CanSet() {
		return false, nil
	}

	valueReflect := reflect.ValueOf(value)
	if valueReflect.Type().AssignableTo(dstElem.Type()) {
		dstElem.Set(valueReflect)
	} else {
		return false, nil
	}

	return true, nil
}

// MGet 批量获取缓存项，支持部分命中和批量回退
func (c *cacherImpl) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	// 检查dstMap是否为指针
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr {
		return nil
	}

	// 检查dstMap是否为map类型
	dstMapElem := dstMapValue.Elem()
	if dstMapElem.Kind() != reflect.Map {
		return nil
	}

	// 先尝试从缓存批量获取
	err := c.store.MGet(ctx, keys, dstMap)
	if err != nil {
		return err
	}

	// 检查哪些键未命中
	missedKeys := make([]string, 0)
	dstMapInterface := dstMapElem.Interface()
	dstMapReflect := reflect.ValueOf(dstMapInterface)

	for _, key := range keys {
		value := dstMapReflect.MapIndex(reflect.ValueOf(key))
		if !value.IsValid() {
			missedKeys = append(missedKeys, key)
		}
	}

	// 如果没有未命中的键，直接返回
	if len(missedKeys) == 0 {
		return nil
	}

	// 如果没有提供回退函数，返回
	if fallback == nil {
		return nil
	}

	// 执行回退函数获取未命中的数据
	fallbackData, err := fallback(ctx, missedKeys)
	if err != nil {
		return err
	}

	// 将回退函数获取的数据存入缓存
	if len(fallbackData) > 0 {
		ttl := time.Duration(0)
		if opts != nil {
			ttl = opts.TTL
		}

		err = c.store.MSet(ctx, fallbackData, ttl)
		if err != nil {
			return err
		}

		// 将回退数据合并到结果中
		for k, v := range fallbackData {
			keyValue := reflect.ValueOf(k)
			valueValue := reflect.ValueOf(v)
			dstMapElem.SetMapIndex(keyValue, valueValue)
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
	// 如果没有提供回退函数，返回
	if fallback == nil {
		return nil
	}

	// 执行回退函数获取数据
	fallbackData, err := fallback(ctx, keys)
	if err != nil {
		return err
	}

	// 将回退函数获取的数据存入缓存
	if len(fallbackData) > 0 {
		ttl := time.Duration(0)
		if opts != nil {
			ttl = opts.TTL
		}

		err = c.store.MSet(ctx, fallbackData, ttl)
		if err != nil {
			return err
		}

		// 检查dstMap是否为指针
		dstMapValue := reflect.ValueOf(dstMap)
		if dstMapValue.Kind() != reflect.Ptr {
			return nil
		}

		// 检查dstMap是否为map类型
		dstMapElem := dstMapValue.Elem()
		if dstMapElem.Kind() != reflect.Map {
			return nil
		}

		// 将回退数据合并到结果中
		for k, v := range fallbackData {
			keyValue := reflect.ValueOf(k)
			valueValue := reflect.ValueOf(v)
			dstMapElem.SetMapIndex(keyValue, valueValue)
		}
	}

	return nil
}
