package cacher

import (
	"context"
	"errors"
	"go-cache/cacher/store"
	"reflect"
	"time"
)

// DefaultCacher Cacher接口的默认实现
type DefaultCacher struct {
	store store.Store
}

// NewCacher 创建新的Cacher实例
func NewCacher(store store.Store) Cacher {
	return &DefaultCacher{
		store: store,
	}
}

// Get 获取单个缓存项，缓存未命中时执行回退函数并缓存结果
func (c *DefaultCacher) Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error) {
	// 验证目标变量
	if err := ValidateDestination(dst); err != nil {
		return false, &CacheError{Op: "get", Key: key, Err: err}
	}

	// 尝试从缓存获取
	found, err := c.store.Get(ctx, key, dst)
	if err != nil {
		return false, &CacheError{Op: "get", Key: key, Err: err}
	}

	// 缓存命中，直接返回
	if found {
		return true, nil
	}

	// 缓存未命中，执行回退函数
	if fallback == nil {
		return false, nil
	}

	value, found, err := fallback(ctx, key)
	if err != nil {
		return false, &CacheError{Op: "fallback", Key: key, Err: err}
	}

	if !found {
		return false, nil
	}

	// 将回退函数的结果赋值给目标变量
	if err := assignValue(dst, value); err != nil {
		return false, &CacheError{Op: "assign", Key: key, Err: err}
	}

	// 将结果写入缓存
	if err := c.setCacheValue(ctx, key, value, opts); err != nil {
		// 缓存写入失败不影响数据获取，只记录错误
		// 可以根据需要决定是否返回错误
	}

	return true, nil
}

// MGet 批量获取缓存项，支持部分命中和批量回退
func (c *DefaultCacher) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	if len(keys) == 0 {
		return nil
	}

	// 验证目标map
	mapValue, elemType, err := ValidateDestinationMap(dstMap)
	if err != nil {
		return &CacheError{Op: "mget", Err: err}
	}

	// 初始化目标map（如果为nil）
	if mapValue.IsNil() {
		mapValue.Set(reflect.MakeMap(mapValue.Type()))
	}

	// 从缓存批量获取
	err = c.store.MGet(ctx, keys, dstMap)
	if err != nil {
		return &CacheError{Op: "mget", Err: err}
	}

	// 检查哪些键未命中
	missedKeys := make([]string, 0)
	for _, key := range keys {
		if !mapValue.MapIndex(reflect.ValueOf(key)).IsValid() {
			missedKeys = append(missedKeys, key)
		}
	}

	// 如果没有未命中的键，直接返回
	if len(missedKeys) == 0 {
		return nil
	}

	// 如果没有回退函数，直接返回
	if fallback == nil {
		return nil
	}

	// 执行批量回退
	fallbackData, err := fallback(ctx, missedKeys)
	if err != nil {
		return &CacheError{Op: "batch_fallback", Err: err}
	}

	// 处理回退数据
	cacheItems := make(map[string]interface{})
	for key, value := range fallbackData {
		// 验证值类型
		if !isAssignableToType(value, elemType) {
			continue // 跳过类型不匹配的值
		}

		// 设置到目标map
		valueReflect := reflect.ValueOf(value)
		mapValue.SetMapIndex(reflect.ValueOf(key), valueReflect)

		// 准备缓存数据
		cacheItems[key] = value
	}

	// 批量写入缓存
	if len(cacheItems) > 0 {
		ttl := time.Duration(0)
		if opts != nil {
			ttl = opts.TTL
		}

		if err := c.store.MSet(ctx, cacheItems, ttl); err != nil {
			// 缓存写入失败不影响数据获取
		}
	}

	return nil
}

// MDelete 批量清除缓存项
func (c *DefaultCacher) MDelete(ctx context.Context, keys []string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	count, err := c.store.Del(ctx, keys...)
	if err != nil {
		return 0, &CacheError{Op: "mdelete", Err: err}
	}

	return count, nil
}

// MRefresh 批量强制刷新缓存项
func (c *DefaultCacher) MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	if len(keys) == 0 {
		return nil
	}

	// 先删除现有缓存
	_, err := c.MDelete(ctx, keys)
	if err != nil {
		return err
	}

	// 重新获取数据
	return c.MGet(ctx, keys, dstMap, fallback, opts)
}

// GetStore 获取底层Store实例
func (c *DefaultCacher) GetStore() store.Store {
	return c.store
}

// 辅助方法：设置单个缓存值
func (c *DefaultCacher) setCacheValue(ctx context.Context, key string, value interface{}, opts *CacheOptions) error {
	items := map[string]interface{}{key: value}
	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}
	return c.store.MSet(ctx, items, ttl)
}

// 辅助函数：将值赋给目标变量
func assignValue(dst interface{}, value interface{}) error {
	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() != reflect.Ptr {
		return errors.New("destination must be a pointer")
	}

	dstElem := dstValue.Elem()
	srcValue := reflect.ValueOf(value)

	// 检查类型兼容性
	if !srcValue.Type().AssignableTo(dstElem.Type()) {
		return errors.New("value type not assignable to destination type")
	}

	dstElem.Set(srcValue)
	return nil
}

// 辅助函数：检查值是否可以赋给指定类型
func isAssignableToType(value interface{}, targetType reflect.Type) bool {
	if value == nil {
		// nil可以赋给指针、切片、映射、通道、函数、接口
		switch targetType.Kind() {
		case reflect.Ptr, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func, reflect.Interface:
			return true
		default:
			return false
		}
	}

	return reflect.TypeOf(value).AssignableTo(targetType)
}
