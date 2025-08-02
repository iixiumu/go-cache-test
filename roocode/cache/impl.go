package cache

import (
	"context"
	"reflect"
	"time"

	"github.com/xiumu/go-cache/store"
)

// cacheImpl 是Cacher接口的实现
type cacheImpl struct {
	store store.Store
}

// New 创建一个新的缓存实例
func New(store store.Store) Cacher {
	return &cacheImpl{
		store: store,
	}
}

// Get 获取单个缓存项，缓存未命中时执行回退函数并缓存结果
func (c *cacheImpl) Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error) {
	// 先尝试从缓存中获取
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
	value, found, err := fallback(ctx, key)
	if err != nil || !found {
		return found, err
	}

	// 将回退函数获取的数据存入缓存
	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}

	err = c.store.MSet(ctx, map[string]interface{}{key: value}, ttl)
	if err != nil {
		// 如果缓存失败，仍然返回回退函数获取的数据
		// 这里我们使用反射将value赋值给dst
		dstValue := reflect.ValueOf(dst)
		if dstValue.Kind() != reflect.Ptr || dstValue.IsNil() {
			return true, nil // 无法赋值，但数据是有效的
		}

		valueReflect := reflect.ValueOf(value)
		dstElem := dstValue.Elem()

		if dstElem.Type().AssignableTo(valueReflect.Type()) {
			dstElem.Set(valueReflect)
		} else if valueReflect.Type().ConvertibleTo(dstElem.Type()) {
			dstElem.Set(valueReflect.Convert(dstElem.Type()))
		}

		return true, nil
	}

	// 使用反射将value赋值给dst
	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() != reflect.Ptr || dstValue.IsNil() {
		return true, nil // 无法赋值，但数据是有效的
	}

	valueReflect := reflect.ValueOf(value)
	dstElem := dstValue.Elem()

	if dstElem.Type().AssignableTo(valueReflect.Type()) {
		dstElem.Set(valueReflect)
	} else if valueReflect.Type().ConvertibleTo(dstElem.Type()) {
		dstElem.Set(valueReflect.Convert(dstElem.Type()))
	}

	return true, nil
}

// MGet 批量获取缓存项，支持部分命中和批量回退
func (c *cacheImpl) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
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
	if dstMapElem.Kind() != reflect.Map {
		return nil
	}

	// 找出未命中的键
	missedKeys := make([]string, 0)
	hitKeys := make(map[string]bool)

	// 遍历dstMap中的键
	for _, key := range keys {
		mapKey := reflect.ValueOf(key)
		if dstMapElem.MapIndex(mapKey).IsValid() {
			hitKeys[key] = true
		} else {
			missedKeys = append(missedKeys, key)
		}
	}

	// 如果所有键都命中了，直接返回
	if len(missedKeys) == 0 {
		return nil
	}

	// 如果没有回退函数，只返回缓存中找到的数据
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
			// 即使缓存失败，也要将数据合并到结果中
		}

		// 将回退数据合并到dstMap中
		for k, v := range fallbackData {
			mapKey := reflect.ValueOf(k)
			mapValue := reflect.ValueOf(v)
			dstMapElem.SetMapIndex(mapKey, mapValue)
		}
	}

	return nil
}

// MDelete 批量清除缓存项
func (c *cacheImpl) MDelete(ctx context.Context, keys []string) (int64, error) {
	return c.store.Del(ctx, keys...)
}

// MRefresh 批量强制刷新缓存项
func (c *cacheImpl) MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	// 不管缓存中是否有数据，都执行回退函数刷新数据
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

		// 将回退数据合并到dstMap中
		dstMapValue := reflect.ValueOf(dstMap)
		if dstMapValue.Kind() != reflect.Ptr || dstMapValue.IsNil() {
			return nil
		}

		dstMapElem := dstMapValue.Elem()
		if dstMapElem.Kind() != reflect.Map {
			return nil
		}

		for k, v := range fallbackData {
			mapKey := reflect.ValueOf(k)
			mapValue := reflect.ValueOf(v)
			dstMapElem.SetMapIndex(mapKey, mapValue)
		}
	}

	return nil
}
