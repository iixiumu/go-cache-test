package cache

import (
	"context"
	"reflect"
	"time"
)

// DefaultTTL 默认缓存过期时间
const DefaultTTL = 5 * time.Minute

// cacheImpl Cacher接口的实现
type cacheImpl struct {
	store Store
}

// NewCacher 创建一个新的Cacher实例
func NewCacher(store Store) Cacher {
	return &cacheImpl{store: store}
}

// Get 获取单个缓存项，缓存未命中时执行回退函数并缓存结果
func (c *cacheImpl) Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error) {
	// 参数验证
	if dst == nil {
		return false, &InvalidArgumentError{"dst cannot be nil"}
	}
	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() != reflect.Ptr || dstValue.IsNil() {
		return false, &InvalidArgumentError{"dst must be a non-nil pointer"}
	}

	// 尝试从缓存获取
	found, err := c.store.Get(ctx, key, dst)
	if err != nil {
		return false, err
	}
	if found {
		return true, nil
	}

	// 缓存未命中，执行回退函数
	if fallback == nil {
		return false, nil
	}

	fallbackValue, fallbackFound, err := fallback(ctx, key)
	if err != nil {
		return false, err
	}
	if !fallbackFound {
		return false, nil
	}

	// 将回退函数的结果写入dst
	dstValue.Elem().Set(reflect.ValueOf(fallbackValue))

	// 缓存回退函数的结果
	ttl := DefaultTTL
	if opts != nil && opts.TTL > 0 {
		ttl = opts.TTL
	}
	err = c.store.MSet(ctx, map[string]interface{}{key: fallbackValue}, ttl)
	if err != nil {
		// 即使缓存失败，也返回回退函数的结果
		return true, nil
	}

	return true, nil
}

// MGet 批量获取缓存项，支持部分命中和批量回退
func (c *cacheImpl) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	// 参数验证
	if len(keys) == 0 {
		return nil
	}
	if dstMap == nil {
		return &InvalidArgumentError{"dstMap cannot be nil"}
	}
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr || dstMapValue.IsNil() {
		return &InvalidArgumentError{"dstMap must be a non-nil pointer"}
	}
	dstMapElem := dstMapValue.Elem()
	if dstMapElem.Kind() != reflect.Map {
		return &InvalidArgumentError{"dstMap must be a pointer to a map"}
	}

	// 初始化目标map
	if dstMapElem.IsNil() {
		dstMapElem.Set(reflect.MakeMap(dstMapElem.Type()))
	}

	// 从缓存批量获取
	err := c.store.MGet(ctx, keys, dstMap)
	if err != nil {
		return err
	}

	// 找出未命中的键
	missedKeys := make([]string, 0, len(keys))
	hitKeys := make(map[string]bool)
	dstMapIter := dstMapElem.MapRange()
	for dstMapIter.Next() {
		hitKeys[dstMapIter.Key().String()] = true
	}
	for _, key := range keys {
		if !hitKeys[key] {
			missedKeys = append(missedKeys, key)
		}
	}

	// 如果没有未命中的键，直接返回
	if len(missedKeys) == 0 {
		return nil
	}

	// 执行批量回退函数
	if fallback == nil {
		return nil
	}
	fallbackValues, err := fallback(ctx, missedKeys)
	if err != nil {
		return err
	}

	// 将回退函数的结果写入目标map
	for key, value := range fallbackValues {
		dstMapElem.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
	}

	// 缓存回退函数的结果
	if len(fallbackValues) > 0 {
		ttl := DefaultTTL
		if opts != nil && opts.TTL > 0 {
			ttl = opts.TTL
		}
		err = c.store.MSet(ctx, fallbackValues, ttl)
		if err != nil {
			// 即使缓存失败，也返回回退函数的结果
			return nil
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
	// 参数验证
	if len(keys) == 0 {
		return nil
	}
	if dstMap == nil {
		return &InvalidArgumentError{"dstMap cannot be nil"}
	}
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr || dstMapValue.IsNil() {
		return &InvalidArgumentError{"dstMap must be a non-nil pointer"}
	}
	dstMapElem := dstMapValue.Elem()
	if dstMapElem.Kind() != reflect.Map {
		return &InvalidArgumentError{"dstMap must be a pointer to a map"}
	}

	// 初始化目标map
	if dstMapElem.IsNil() {
		dstMapElem.Set(reflect.MakeMap(dstMapElem.Type()))
	}

	// 执行批量回退函数获取最新数据
	if fallback == nil {
		return &InvalidArgumentError{"fallback cannot be nil for MRefresh"}
	}
	refreshedValues, err := fallback(ctx, keys)
	if err != nil {
		return err
	}

	// 将回退函数的结果写入目标map
	for key, value := range refreshedValues {
		dstMapElem.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
	}

	// 缓存回退函数的结果
	if len(refreshedValues) > 0 {
		ttl := DefaultTTL
		if opts != nil && opts.TTL > 0 {
			ttl = opts.TTL
		}
		err = c.store.MSet(ctx, refreshedValues, ttl)
		if err != nil {
			// 即使缓存失败，也返回回退函数的结果
			return nil
		}
	}

	return nil
}

// InvalidArgumentError 无效参数错误
type InvalidArgumentError struct {
	Message string
}

func (e *InvalidArgumentError) Error() string {
	return e.Message
}