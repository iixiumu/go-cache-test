package cacher

import (
	"context"
	"errors"
	"reflect"
	"time"

	"github.com/xiumu/git/me/go-cache/trae/pkg/store"
)

// cacheImpl 是Cacher接口的具体实现
type cacheImpl struct {
	store store.Store
}

// NewCacher 创建一个新的Cacher实例
func NewCacher(s store.Store) Cacher {
	return &cacheImpl{
		store: s,
	}
}

// Get 获取单个缓存项，缓存未命中时执行回退函数并缓存结果
func (c *cacheImpl) Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error) {
	// 检查参数
	if dst == nil || reflect.TypeOf(dst).Kind() != reflect.Ptr {
		return false, errors.New("dst must be a pointer")
	}

	// 从缓存获取
	found, err := c.store.Get(ctx, key, dst)
	if err != nil {
		return false, err
	}

	// 缓存命中
	if found {
		return true, nil
	}

	// 缓存未命中，调用回退函数
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

	// 设置TTL
	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}

	// 缓存结果
	items := map[string]interface{}{
		key: value,
	}
	if err := c.store.MSet(ctx, items, ttl); err != nil {
		// 记录错误但不阻止返回结果
		return true, err
	}

	// 将值设置到dst
	if err := setValue(dst, value); err != nil {
		return true, err
	}

	return true, nil
}

// setValue 将值设置到目标指针
func setValue(dst interface{}, value interface{}) error {
	// 检查dst是否为指针
	if dst == nil || reflect.TypeOf(dst).Kind() != reflect.Ptr {
		return errors.New("dst must be a pointer")
	}

	// 获取dst的反射值
	dstValue := reflect.ValueOf(dst).Elem()
	valueValue := reflect.ValueOf(value)

	// 确保类型匹配
	if !valueValue.Type().AssignableTo(dstValue.Type()) {
		// 尝试转换类型
		if !valueValue.Type().ConvertibleTo(dstValue.Type()) {
			return errors.New("cannot convert value to dst type")
		}
		valueValue = valueValue.Convert(dstValue.Type())
	}

	// 设置值
	dstValue.Set(valueValue)
	return nil


	return true, nil
}

// MGet 批量获取缓存项，支持部分命中和批量回退
func (c *cacheImpl) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	// 检查参数
	if dstMap == nil || reflect.TypeOf(dstMap).Kind() != reflect.Ptr {
		return errors.New("dstMap must be a pointer")
	}

	mapType := reflect.TypeOf(dstMap).Elem()
	if mapType.Kind() != reflect.Map {
		return errors.New("dstMap must be a pointer to a map")
	}

	// 创建结果map
	resultMap := reflect.MakeMap(mapType)
	reflect.ValueOf(dstMap).Elem().Set(resultMap)

	// 从缓存批量获取
	cacheResults := make(map[string]interface{})
	if err := c.store.MGet(ctx, keys, &cacheResults); err != nil {
		return err
	}

	// 找出未命中的键
	var missingKeys []string
	for _, key := range keys {
		if _, found := cacheResults[key]; !found {
			missingKeys = append(missingKeys, key)
		}
	}

	// 如果有未命中的键且提供了回退函数，则调用回退函数
	if len(missingKeys) > 0 && fallback != nil {
		fallbackResults, err := fallback(ctx, missingKeys)
		if err != nil {
			return err
		}

		// 设置TTL
		ttl := time.Duration(0)
		if opts != nil {
			ttl = opts.TTL
		}

		// 缓存回退结果
		if ttl > 0 {
			if err := c.store.MSet(ctx, fallbackResults, ttl); err != nil {
				// 记录错误但不阻止返回结果
			}
		}

		// 合并回退结果到缓存结果
		for k, v := range fallbackResults {
			cacheResults[k] = v
		}
	}

	// 将结果设置到dstMap
	mapValue := reflect.ValueOf(dstMap).Elem()
	for k, v := range cacheResults {
		keyValue := reflect.ValueOf(k)
		valueValue := reflect.ValueOf(v)

		// 确保类型匹配
		if !valueValue.Type().AssignableTo(mapType.Elem()) {
			// 尝试转换类型
			if !valueValue.Type().ConvertibleTo(mapType.Elem()) {
				continue
			}
			valueValue = valueValue.Convert(mapType.Elem())
		}

		mapValue.SetMapIndex(keyValue, valueValue)
	}

	return nil
}

// MDelete 批量清除缓存项
func (c *cacheImpl) MDelete(ctx context.Context, keys []string) (int64, error) {
	return c.store.Del(ctx, keys)
}

// MRefresh 批量强制刷新缓存项
func (c *cacheImpl) MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	// 检查参数
	if dstMap == nil || reflect.TypeOf(dstMap).Kind() != reflect.Ptr {
		return errors.New("dstMap must be a pointer")
	}

	mapType := reflect.TypeOf(dstMap).Elem()
	if mapType.Kind() != reflect.Map {
		return errors.New("dstMap must be a pointer to a map")
	}

	// 首先删除缓存中的项
	if _, err := c.store.Del(ctx, keys...); err != nil {
		return err
	}

	// 如果没有提供回退函数，直接返回空结果
	if fallback == nil {
		resultMap := reflect.MakeMap(mapType)
		reflect.ValueOf(dstMap).Elem().Set(resultMap)
		return nil
	}

	// 调用回退函数获取最新值
	fallbackResults, err := fallback(ctx, keys)
	if err != nil {
		return err
	}

	// 设置TTL
	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}

	// 缓存新值
	if err := c.store.MSet(ctx, fallbackResults, ttl); err != nil {
		// 记录错误但不阻止返回结果
	}

	// 将结果设置到dstMap
	mapValue := reflect.ValueOf(dstMap).Elem()
	resultMap := reflect.MakeMap(mapType)
	for k, v := range fallbackResults {
		keyValue := reflect.ValueOf(k)
		valueValue := reflect.ValueOf(v)

		// 确保类型匹配
		if !valueValue.Type().AssignableTo(mapType.Elem()) {
			// 尝试转换类型
			if !valueValue.Type().ConvertibleTo(mapType.Elem()) {
				continue
			}
			valueValue = valueValue.Convert(mapType.Elem())
		}

		resultMap.SetMapIndex(keyValue, valueValue)
	}

	mapValue.Set(resultMap)

	return nil
}

// setValue 使用反射将值设置到目标指针
func setValue(dst interface{}, value interface{}) error {
	dstValue := reflect.ValueOf(dst).Elem()
	valueValue := reflect.ValueOf(value)

	// 确保类型匹配
	if !valueValue.Type().AssignableTo(dstValue.Type()) {
		// 尝试转换类型
		if !valueValue.Type().ConvertibleTo(dstValue.Type()) {
			return errors.New("cannot assign value to dst: type mismatch")
		}
		valueValue = valueValue.Convert(dstValue.Type())
	}

	dstValue.Set(valueValue)
	return nil
}