package cache

import (
	"context"
	"errors"
	"reflect"
	"time"

	"github.com/xiumu/go-cache/store"
)

// DefaultCacheOptions 默认缓存选项
var DefaultCacheOptions = &CacheOptions{
	TTL: 10 * time.Minute,
}

// cacheImpl Cacher接口的实现
type cacheImpl struct {
	store store.Store
}

// New 创建一个新的Cacher实例
func New(store store.Store) Cacher {
	return &cacheImpl{
		store: store,
	}
}

// Get 获取单个缓存项，缓存未命中时执行回退函数并缓存结果
func (c *cacheImpl) Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error) {
	// 使用默认选项如果未提供
	if opts == nil {
		opts = DefaultCacheOptions
	}

	// 先尝试从缓存获取
	found, err := c.store.Get(ctx, key, dst)
	if err != nil {
		return false, err
	}

	// 如果找到了直接返回
	if found {
		return true, nil
	}

	// 如果没有回退函数，直接返回未找到
	if fallback == nil {
		return false, nil
	}

	// 执行回退函数获取数据
	value, found, err := fallback(ctx, key)
	if err != nil {
		return false, err
	}

	// 如果回退函数未找到数据，直接返回
	if !found {
		return false, nil
	}

	// 将回退函数获取到的数据存入缓存
	items := map[string]interface{}{key: value}
	err = c.store.MSet(ctx, items, opts.TTL)
	if err != nil {
		return false, err
	}

	// 将值赋给dst
	// 这里需要使用反射来处理dst是指针的情况
	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return false, errors.New("dst must be a non-nil pointer")
	}

	// 使用反射设置dst的值
	val := reflect.ValueOf(value)
	if val.Type().AssignableTo(rv.Elem().Type()) {
		rv.Elem().Set(val)
	} else {
		return false, errors.New("value type mismatch")
	}

	return true, nil
}

// MGet 批量获取缓存项，支持部分命中和批量回退
func (c *cacheImpl) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	// 使用默认选项如果未提供
	if opts == nil {
		opts = DefaultCacheOptions
	}

	// 先尝试从缓存批量获取
	err := c.store.MGet(ctx, keys, dstMap)
	if err != nil {
		return err
	}

	// 检查哪些键未命中，需要执行回退函数
	// 获取dstMap的反射值
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr || dstMapValue.IsNil() {
		return errors.New("dstMap must be a non-nil pointer to a map")
	}

	// 解引用指针获取实际的map
	dstMapElem := dstMapValue.Elem()
	if dstMapElem.Kind() != reflect.Map {
		return errors.New("dstMap must be a pointer to a map")
	}

	// 检查哪些键已经存在
	hitKeys := make(map[string]bool)
	for _, key := range keys {
		// 检查map中是否已存在该键
		mapKey := reflect.ValueOf(key)
		mapValue := dstMapElem.MapIndex(mapKey)
		if mapValue.IsValid() {
			hitKeys[key] = true
		}
	}

	// 找出未命中的键
	var missKeys []string
	for _, key := range keys {
		if !hitKeys[key] {
			missKeys = append(missKeys, key)
		}
	}

	// 如果没有未命中的键，直接返回
	if len(missKeys) == 0 {
		return nil
	}

	// 如果没有回退函数，直接返回已有的数据
	if fallback == nil {
		return nil
	}

	// 执行回退函数获取未命中的数据
	values, err := fallback(ctx, missKeys)
	if err != nil {
		return err
	}

	// 将回退函数获取到的数据存入缓存
	if len(values) > 0 {
		err = c.store.MSet(ctx, values, opts.TTL)
		if err != nil {
			return err
		}

		// 将新获取的数据合并到dstMap中
		for key, value := range values {
			mapKey := reflect.ValueOf(key)
			mapValue := reflect.ValueOf(value)
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
	// 使用默认选项如果未提供
	if opts == nil {
		opts = DefaultCacheOptions
	}

	// 直接执行回退函数获取数据
	if fallback == nil {
		return errors.New("fallback function is required for refresh")
	}

	values, err := fallback(ctx, keys)
	if err != nil {
		return err
	}

	// 将获取到的数据存入缓存
	err = c.store.MSet(ctx, values, opts.TTL)
	if err != nil {
		return err
	}

	// 将值赋给dstMap
	// 获取dstMap的反射值
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr || dstMapValue.IsNil() {
		return errors.New("dstMap must be a non-nil pointer to a map")
	}

	// 解引用指针获取实际的map
	dstMapElem := dstMapValue.Elem()
	if dstMapElem.Kind() != reflect.Map {
		return errors.New("dstMap must be a pointer to a map")
	}

	// 清空现有map
	dstMapElem.Set(reflect.MakeMap(dstMapElem.Type()))

	// 将新获取的数据填充到dstMap中
	for key, value := range values {
		mapKey := reflect.ValueOf(key)
		mapValue := reflect.ValueOf(value)
		dstMapElem.SetMapIndex(mapKey, mapValue)
	}

	return nil
}