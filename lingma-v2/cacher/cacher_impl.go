package cacher

import (
	"context"
	"reflect"
	"time"

	"github.com/davecgh/go-spew/spew"
	"go-cache/cacher/store"
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
	// 尝试从缓存中获取
	dstPtr := reflect.New(reflect.TypeOf(dst).Elem()).Interface()
	found, err := c.store.Get(ctx, key, dstPtr)
	if err != nil {
		return false, err
	}

	if found {
		// 缓存命中，将值复制到目标变量
		reflect.ValueOf(dst).Elem().Set(reflect.ValueOf(dstPtr).Elem())
		return true, nil
	}

	// 缓存未命中，执行回退函数
	if fallback == nil {
		return false, nil
	}

	value, found, err := fallback(ctx, key)
	if err != nil || !found {
		return found, err
	}

	// 将回退函数的结果缓存
	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}

	err = c.store.MSet(ctx, map[string]interface{}{key: value}, ttl)
	if err != nil {
		// 如果缓存失败，仍然返回回退函数的结果
		spew.Dump("Failed to cache value:", err)
	}

	// 将值复制到目标变量
	reflect.ValueOf(dst).Elem().Set(reflect.ValueOf(value))
	return true, nil
}

// MGet 批量获取缓存项，支持部分命中和批量回退
func (c *cacherImpl) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	// 使用反射处理目标 map
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr || dstMapValue.Elem().Kind() != reflect.Map {
		return &CacheError{"dstMap must be a pointer to a map"}
	}

	mapValue := dstMapValue.Elem()
	mapValue.Set(reflect.MakeMap(mapValue.Type()))

	// 批量从缓存获取
	err := c.store.MGet(ctx, keys, dstMap)
	if err != nil {
		return err
	}

	// 检查是否有未命中的键
	missedKeys := c.findMissedKeys(keys, dstMap)

	// 如果没有回退函数或没有未命中的键，则直接返回
	if fallback == nil || len(missedKeys) == 0 {
		return nil
	}

	// 执行批量回退函数
	fallbackValues, err := fallback(ctx, missedKeys)
	if err != nil {
		return err
	}

	// 将回退结果添加到结果 map 中
	keyType := mapValue.Type().Key()
	elemType := mapValue.Type().Elem()

	for key, value := range fallbackValues {
		// 检查值类型是否兼容
		valueValue := reflect.ValueOf(value)
		if !valueValue.Type().AssignableTo(elemType) {
			if valueValue.Type().ConvertibleTo(elemType) {
				valueValue = valueValue.Convert(elemType)
			} else {
				continue // 跳过不兼容的类型
			}
		}

		// 添加到结果 map
		mapValue.SetMapIndex(reflect.ValueOf(key).Convert(keyType), valueValue)
	}

	// 缓存回退结果
	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}

	// 批量设置回退结果到缓存
	err = c.store.MSet(ctx, fallbackValues, ttl)
	if err != nil {
		// 如果缓存失败，记录日志但不中断操作
		spew.Dump("Failed to cache fallback values:", err)
	}

	return nil
}

// MDelete 批量清除缓存项
func (c *cacherImpl) MDelete(ctx context.Context, keys []string) (int64, error) {
	return c.store.Del(ctx, keys...)
}

// MRefresh 批量强制刷新缓存项
func (c *cacherImpl) MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	if fallback == nil {
		return &CacheError{"fallback function is required for refresh"}
	}

	// 执行批量回退函数获取最新数据
	fallbackValues, err := fallback(ctx, keys)
	if err != nil {
		return err
	}

	// 将回退结果添加到结果 map 中
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr || dstMapValue.Elem().Kind() != reflect.Map {
		return &CacheError{"dstMap must be a pointer to a map"}
	}

	mapValue := dstMapValue.Elem()
	mapValue.Set(reflect.MakeMap(mapValue.Type()))

	keyType := mapValue.Type().Key()
	elemType := mapValue.Type().Elem()

	for key, value := range fallbackValues {
		// 检查值类型是否兼容
		valueValue := reflect.ValueOf(value)
		if !valueValue.Type().AssignableTo(elemType) {
			if valueValue.Type().ConvertibleTo(elemType) {
				valueValue = valueValue.Convert(elemType)
			} else {
				continue // 跳过不兼容的类型
			}
		}

		// 添加到结果 map
		mapValue.SetMapIndex(reflect.ValueOf(key).Convert(keyType), valueValue)
	}

	// 缓存回退结果
	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}

	return c.store.MSet(ctx, fallbackValues, ttl)
}

// findMissedKeys 查找未命中的键
func (c *cacherImpl) findMissedKeys(keys []string, result interface{}) []string {
	// 使用反射获取结果 map
	resultValue := reflect.ValueOf(result)
	if resultValue.Kind() != reflect.Map {
		return keys
	}

	missed := make([]string, 0)
	for _, key := range keys {
		// 检查键是否在结果中存在
		value := resultValue.MapIndex(reflect.ValueOf(key))
		if !value.IsValid() {
			missed = append(missed, key)
		}
	}

	return missed
}

// CacheError 表示缓存相关的错误
type CacheError struct {
	message string
}

func (e *CacheError) Error() string {
	return e.message
}