package cacher

import (
	"context"
	"reflect"
	"time"

	"go-cache/cacher/store"
)

// cacherImpl 实现Cacher接口
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
	value, found, err := fallback(ctx, key)
	if err != nil {
		return false, err
	}

	// 如果回退函数没有找到数据，返回未找到
	if !found {
		return false, nil
	}

	// 将回退函数获取的数据缓存
	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}

	err = c.store.MSet(ctx, map[string]interface{}{key: value}, ttl)
	if err != nil {
		return false, err
	}

	// 使用反射将值设置到dst
	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() != reflect.Ptr {
		return false, &reflect.ValueError{Method: "Get", Kind: dstValue.Kind()}
	}

	if dstValue.IsNil() {
		return false, &reflect.ValueError{Method: "Get", Kind: dstValue.Kind()}
	}

	dstElem := dstValue.Elem()
	srcValue := reflect.ValueOf(value)

	// 如果类型匹配，直接设置值
	if srcValue.Type().AssignableTo(dstElem.Type()) {
		dstElem.Set(srcValue)
	} else {
		// 尝试转换类型
		if srcValue.Type().ConvertibleTo(dstElem.Type()) {
			dstElem.Set(srcValue.Convert(dstElem.Type()))
		} else {
			return false, &reflect.ValueError{Method: "Get", Kind: dstElem.Kind()}
		}
	}

	return true, nil
}

// MGet 批量获取缓存项，支持部分命中和批量回退
func (c *cacherImpl) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	// 使用反射来设置目标map
	dstValue, err := c.getMapPtrValue(dstMap)
	if err != nil {
		return err
	}

	// 清空目标map
	dstValue.Set(reflect.MakeMap(dstValue.Type()))

	// 从存储中批量获取
	err = c.store.MGet(ctx, keys, dstMap)
	if err != nil {
		return err
	}

	// 检查哪些键没有找到
	foundKeys := make(map[string]bool)
	// 遍历dstMap来找出已找到的键
	dstMapValue := reflect.ValueOf(dstMap).Elem()
	mapRange := dstMapValue.MapRange()
	for mapRange.Next() {
		foundKeys[mapRange.Key().String()] = true
	}

	// 找出未命中的键
	var missingKeys []string
	for _, key := range keys {
		if !foundKeys[key] {
			missingKeys = append(missingKeys, key)
		}
	}

	// 如果没有未命中的键，直接返回
	if len(missingKeys) == 0 {
		return nil
	}

	// 如果没有回退函数，直接返回
	if fallback == nil {
		return nil
	}

	// 执行批量回退函数获取未命中的数据
	fallbackData, err := fallback(ctx, missingKeys)
	if err != nil {
		return err
	}

	// 将回退函数获取的数据缓存
	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}

	if len(fallbackData) > 0 {
		err = c.store.MSet(ctx, fallbackData, ttl)
		if err != nil {
			return err
		}
	}

	// 将回退数据添加到结果中
	for key, value := range fallbackData {
		// 创建新元素
		elemType := dstValue.Type().Elem()
		elemValue := reflect.New(elemType).Elem()

		// 设置值
		srcValue := reflect.ValueOf(value)
		if srcValue.Type().AssignableTo(elemType) {
			elemValue.Set(srcValue)
		} else if srcValue.Type().ConvertibleTo(elemType) {
			elemValue.Set(srcValue.Convert(elemType))
		} else {
			continue
		}

		// 设置map值
		dstValue.SetMapIndex(reflect.ValueOf(key), elemValue)
	}

	return nil
}

// MDelete 批量清除缓存项
func (c *cacherImpl) MDelete(ctx context.Context, keys []string) (int64, error) {
	return c.store.Del(ctx, keys...)
}

// MRefresh 批量强制刷新缓存项
func (c *cacherImpl) MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	// 如果没有回退函数，无法刷新
	if fallback == nil {
		return nil
	}

	// 执行批量回退函数获取数据
	fallbackData, err := fallback(ctx, keys)
	if err != nil {
		return err
	}

	// 将回退函数获取的数据缓存
	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}

	if len(fallbackData) > 0 {
		err = c.store.MSet(ctx, fallbackData, ttl)
		if err != nil {
			return err
		}
	}

	// 使用反射来设置目标map
	dstValue, err := c.getMapPtrValue(dstMap)
	if err != nil {
		return err
	}

	// 清空目标map
	dstValue.Set(reflect.MakeMap(dstValue.Type()))

	// 将数据添加到结果中
	for key, value := range fallbackData {
		// 创建新元素
		elemType := dstValue.Type().Elem()
		elemValue := reflect.New(elemType).Elem()

		// 设置值
		srcValue := reflect.ValueOf(value)
		if srcValue.Type().AssignableTo(elemType) {
			elemValue.Set(srcValue)
		} else if srcValue.Type().ConvertibleTo(elemType) {
			elemValue.Set(srcValue.Convert(elemType))
		} else {
			continue
		}

		// 设置map值
		dstValue.SetMapIndex(reflect.ValueOf(key), elemValue)
	}

	return nil
}

// getMapPtrValue 获取map指针的reflect.Value
func (c *cacherImpl) getMapPtrValue(mapPtr interface{}) (reflect.Value, error) {
	v := reflect.ValueOf(mapPtr)
	if v.Kind() != reflect.Ptr {
		return reflect.Value{}, &reflect.ValueError{Method: "getMapPtrValue", Kind: v.Kind()}
	}
	if v.Elem().Kind() != reflect.Map {
		return reflect.Value{}, &reflect.ValueError{Method: "getMapPtrValue", Kind: v.Elem().Kind()}
	}
	return v.Elem(), nil
}
