package cacher

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"go-cache/cacher/store"
)

// cacherImpl 实现了Cacher接口
type cacherImpl struct {
	store store.Store
}

// NewCacher 创建一个新的Cacher实例
func NewCacher(store store.Store) Cacher {
	return &cacherImpl{
		store: store,
	}
}

// Get 获取单个缓存项，缓存未命中时执行回退函数并缓存结果
func (c *cacherImpl) Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error) {
	// 尝试从存储中获取值
	found, err := c.store.Get(ctx, key, dst)
	if err != nil {
		return false, err
	}

	// 如果找到了值，直接返回
	if found {
		return true, nil
	}

	// 如果没有找到值且没有回退函数，返回未找到
	if fallback == nil {
		return false, nil
	}

	// 执行回退函数获取值
	value, found, err := fallback(ctx, key)
	if err != nil {
		return false, err
	}

	// 如果回退函数没有找到值，返回未找到
	if !found {
		return false, nil
	}

	// 将回退函数返回的值存储到缓存中
	items := map[string]interface{}{key: value}
	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}

	err = c.store.MSet(ctx, items, ttl)
	if err != nil {
		return false, err
	}

	// 使用反射将值复制到目标变量
	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() != reflect.Ptr {
		return false, fmt.Errorf("dst must be a pointer")
	}

	// 获取目标变量的类型
	dstElem := dstValue.Elem()
	valueReflect := reflect.ValueOf(value)

	// 检查类型兼容性
	if !valueReflect.Type().AssignableTo(dstElem.Type()) {
		// 尝试转换类型
		if valueReflect.CanConvert(dstElem.Type()) {
			dstElem.Set(valueReflect.Convert(dstElem.Type()))
		} else {
			return false, fmt.Errorf("cannot assign %T to %T", value, dst)
		}
	} else {
		dstElem.Set(valueReflect)
	}

	return true, nil
}

// MGet 批量获取缓存项，支持部分命中和批量回退
func (c *cacherImpl) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	// 从存储中批量获取值
	err := c.store.MGet(ctx, keys, dstMap)
	if err != nil {
		return err
	}

	// 检查哪些键未命中
	// 使用反射获取dstMap中的键
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr || dstMapValue.Elem().Kind() != reflect.Map {
		return fmt.Errorf("dstMap must be a pointer to a map")
	}

	// 获取map中的键
	existingKeys := make(map[string]bool)
	dstMapElem := dstMapValue.Elem()
	for _, key := range dstMapElem.MapKeys() {
		existingKeys[key.String()] = true
	}

	// 找出未命中的键
	var missingKeys []string
	for _, key := range keys {
		if !existingKeys[key] {
			missingKeys = append(missingKeys, key)
		}
	}

	// 如果没有未命中的键或没有回退函数，直接返回
	if len(missingKeys) == 0 || fallback == nil {
		return nil
	}

	// 执行批量回退函数获取未命中的值
	values, err := fallback(ctx, missingKeys)
	if err != nil {
		return err
	}

	// 将回退函数返回的值存储到缓存中
	if len(values) > 0 {
		ttl := time.Duration(0)
		if opts != nil {
			ttl = opts.TTL
		}

		err = c.store.MSet(ctx, values, ttl)
		if err != nil {
			return err
		}

		// 将新值添加到结果map中
		// 使用反射将值添加到dstMap中
		for key, value := range values {
			keyValue := reflect.ValueOf(key)
			valueValue := reflect.ValueOf(value)
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
	// 如果没有回退函数，直接返回
	if fallback == nil {
		return nil
	}

	// 执行批量回退函数获取值
	values, err := fallback(ctx, keys)
	if err != nil {
		return err
	}

	// 将回退函数返回的值存储到缓存中
	if len(values) > 0 {
		ttl := time.Duration(0)
		if opts != nil {
			ttl = opts.TTL
		}

		err = c.store.MSet(ctx, values, ttl)
		if err != nil {
			return err
		}

		// 将值复制到结果map中
		// 使用反射将值复制到dstMap中
		dstMapValue := reflect.ValueOf(dstMap)
		if dstMapValue.Kind() != reflect.Ptr || dstMapValue.Elem().Kind() != reflect.Map {
			return fmt.Errorf("dstMap must be a pointer to a map")
		}

		dstMapElem := dstMapValue.Elem()
		for key, value := range values {
			keyValue := reflect.ValueOf(key)
			valueValue := reflect.ValueOf(value)
			dstMapElem.SetMapIndex(keyValue, valueValue)
		}
	}

	return nil
}
