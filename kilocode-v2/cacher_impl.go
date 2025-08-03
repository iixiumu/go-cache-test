package cacher

import (
	"context"
	"reflect"
	"time"

	"go-cache/store"
)

// cacherImpl 是 Cacher 接口的实现
type cacherImpl struct {
	store store.Store
}

// NewCacher 创建一个新的 Cacher 实例
func NewCacher(store store.Store) Cacher {
	return &cacherImpl{
		store: store,
	}
}

// Get 获取单个缓存项，缓存未命中时执行回退函数并缓存结果
func (c *cacherImpl) Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error) {
	// 首先尝试从存储中获取
	found, err := c.store.Get(ctx, key, dst)
	if err != nil {
		return false, err
	}

	// 如果找到了，直接返回
	if found {
		return true, nil
	}

	// 如果没有找到且没有回退函数，返回未找到
	if fallback == nil {
		return false, nil
	}

	// 执行回退函数获取数据
	value, fallbackFound, err := fallback(ctx, key)
	if err != nil {
		return false, err
	}

	// 如果回退函数没有找到数据，返回未找到
	if !fallbackFound {
		return false, nil
	}

	// 确定 TTL
	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}

	// 将回退函数返回的数据存储到缓存中
	err = c.store.MSet(ctx, map[string]interface{}{key: value}, ttl)
	if err != nil {
		return true, err // 返回 true 因为数据确实存在，只是存储时出错
	}

	// 将值复制到 dst
	return true, copyValue(dst, value)
}

// MGet 批量获取缓存项，支持部分命中和批量回退
func (c *cacherImpl) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	// 验证 dstMap 是一个指向 map 的指针
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr || dstMapValue.Elem().Kind() != reflect.Map {
		return &InvalidTypeError{Message: "dstMap must be a pointer to a map"}
	}

	// 初始化目标 map
	dstMapValue.Elem().Set(reflect.MakeMap(dstMapValue.Elem().Type()))

	// 从存储中批量获取
	err := c.store.MGet(ctx, keys, dstMap)
	if err != nil {
		return err
	}

	// 检查哪些键未命中
	missedKeys := c.findMissedKeys(keys, dstMapValue.Elem())

	// 如果没有未命中的键或者没有回退函数，直接返回
	if len(missedKeys) == 0 || fallback == nil {
		return nil
	}

	// 执行批量回退函数获取未命中的数据
	fallbackValues, err := fallback(ctx, missedKeys)
	if err != nil {
		return err
	}

	// 确定 TTL
	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}

	// 将回退函数返回的数据存储到缓存中
	if len(fallbackValues) > 0 {
		err = c.store.MSet(ctx, fallbackValues, ttl)
		if err != nil {
			return err
		}

		// 将回退的数据合并到结果中
		for k, v := range fallbackValues {
			mapKey := reflect.ValueOf(k)
			mapValue := reflect.ValueOf(v)
			dstMapValue.Elem().SetMapIndex(mapKey, mapValue)
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
	// 验证 dstMap 是一个指向 map 的指针
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr || dstMapValue.Elem().Kind() != reflect.Map {
		return &InvalidTypeError{Message: "dstMap must be a pointer to a map"}
	}

	// 初始化目标 map
	dstMapValue.Elem().Set(reflect.MakeMap(dstMapValue.Elem().Type()))

	// 如果没有回退函数，直接返回
	if fallback == nil {
		return nil
	}

	// 执行批量回退函数获取数据
	values, err := fallback(ctx, keys)
	if err != nil {
		return err
	}

	// 确定 TTL
	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}

	// 将数据存储到缓存中
	if len(values) > 0 {
		err = c.store.MSet(ctx, values, ttl)
		if err != nil {
			return err
		}

		// 将数据合并到结果中
		for k, v := range values {
			mapKey := reflect.ValueOf(k)
			mapValue := reflect.ValueOf(v)
			dstMapValue.Elem().SetMapIndex(mapKey, mapValue)
		}
	}

	return nil
}

// findMissedKeys 查找未命中的键
func (c *cacherImpl) findMissedKeys(keys []string, resultMap reflect.Value) []string {
	missed := make([]string, 0)
	for _, key := range keys {
		if !resultMap.MapIndex(reflect.ValueOf(key)).IsValid() {
			missed = append(missed, key)
		}
	}
	return missed
}

// copyValue 将 src 的值复制到 dst
func copyValue(dst interface{}, src interface{}) error {
	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() != reflect.Ptr {
		return &InvalidTypeError{Message: "dst must be a pointer"}
	}

	srcValue := reflect.ValueOf(src)
	dstValue.Elem().Set(srcValue)

	return nil
}

// InvalidTypeError 无效类型错误
type InvalidTypeError struct {
	Message string
}

func (e *InvalidTypeError) Error() string {
	return e.Message
}
