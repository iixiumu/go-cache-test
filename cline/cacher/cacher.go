package cacher

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/example/go-cache/store"
)

// FallbackFunc 回退函数类型
// 当缓存未命中时执行，用于从数据源获取数据
// key: 请求的键
// 返回: 获取到的值, 是否找到, 错误信息
type FallbackFunc func(ctx context.Context, key string) (interface{}, bool, error)

// BatchFallbackFunc 批量回退函数类型
// 当批量缓存部分未命中时执行，用于从数据源批量获取数据
// keys: 未命中的键列表
// 返回: 键值映射, 错误信息
type BatchFallbackFunc func(ctx context.Context, keys []string) (map[string]interface{}, error)

// CacheOptions 缓存选项
type CacheOptions struct {
	// TTL 缓存过期时间，0表示永不过期
	TTL time.Duration
}

// Cacher 高级缓存接口，提供带回退机制的缓存操作
type Cacher interface {
	// Get 获取单个缓存项，缓存未命中时执行回退函数并缓存结果
	// key: 键名
	// dst: 目标变量的指针，用于接收值
	// fallback: 缓存未命中时的回退函数
	// opts: 缓存选项，可以为nil使用默认选项
	// 返回: 是否找到值（包括从回退函数获取）, 错误信息
	Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error)

	// MGet 批量获取缓存项，支持部分命中和批量回退
	// keys: 要获取的键列表
	// dstMap: 目标map的指针，用于接收结果，类型为*map[string]T
	// fallback: 批量回退函数，处理未命中的键
	// opts: 缓存选项，可以为nil使用默认选项
	// 返回: 错误信息
	MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error

	// MDelete 批量清除缓存项
	// keys: 要删除的键列表
	// 返回: 实际删除的键数量, 错误信息
	MDelete(ctx context.Context, keys []string) (int64, error)

	// MRefresh 批量强制刷新缓存项
	// keys: 要刷新的键列表
	// dstMap: 目标map的指针，用于接收结果，类型为*map[string]T
	// fallback: 批量回退函数
	// opts: 缓存选项，可以为nil使用默认选项
	// 返回: 错误信息
	MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error
}

// NewCacher 创建一个新的Cacher实例
func NewCacher(store store.Store) Cacher {
	return &cacherImpl{
		store: store,
	}
}

// cacherImpl Cacher接口的实现
type cacherImpl struct {
	store store.Store
}

// Get 获取单个缓存项，缓存未命中时执行回退函数并缓存结果
func (c *cacherImpl) Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error) {
	// 首先尝试从缓存中获取
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
	if err != nil || !found {
		return found, err
	}

	// 将值赋给dst
	// 使用反射来处理不同的类型
	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return false, fmt.Errorf("dst must be a non-nil pointer")
	}

	rv = rv.Elem()
	if !rv.CanSet() {
		return false, fmt.Errorf("cannot set dst value")
	}

	valRV := reflect.ValueOf(value)
	if !valRV.Type().AssignableTo(rv.Type()) {
		return false, fmt.Errorf("cannot assign %T to %T", value, dst)
	}

	rv.Set(valRV)

	// 将获取到的数据存入缓存
	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}

	err = c.store.MSet(ctx, map[string]interface{}{key: value}, ttl)
	if err != nil {
		return true, err // 返回true因为数据已获取，但缓存存储失败
	}

	return true, nil
}

// MGet 批量获取缓存项，支持部分命中和批量回退
func (c *cacherImpl) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	// 批量从缓存中获取
	err := c.store.MGet(ctx, keys, dstMap)
	if err != nil {
		return err
	}

	// 如果没有提供回退函数，直接返回缓存结果
	if fallback == nil {
		return nil
	}

	// 检查哪些键未命中
	// 使用反射来检查dstMap中哪些键不存在
	rv := reflect.ValueOf(dstMap)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("dstMap must be a non-nil pointer")
	}

	rv = rv.Elem()
	if rv.Kind() != reflect.Map {
		return fmt.Errorf("dstMap must be a pointer to a map")
	}

	// 收集未命中的键
	var missedKeys []string
	dstMapKeys := rv.MapKeys()
	dstMapKeySet := make(map[string]bool)
	for _, keyRV := range dstMapKeys {
		dstMapKeySet[keyRV.String()] = true
	}

	for _, key := range keys {
		if !dstMapKeySet[key] {
			missedKeys = append(missedKeys, key)
		}
	}

	// 如果没有未命中的键，直接返回
	if len(missedKeys) == 0 {
		return nil
	}

	// 执行批量回退函数获取未命中的数据
	values, err := fallback(ctx, missedKeys)
	if err != nil {
		return err
	}

	// 将获取到的数据存入缓存
	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}

	if len(values) > 0 {
		err = c.store.MSet(ctx, values, ttl)
		if err != nil {
			return err
		}

		// 更新dstMap
		// 获取map的键和值类型
		mapType := rv.Type()
		keyType := mapType.Key()
		valueType := mapType.Elem()

		for key, value := range values {
			// 创建键和值的反射值
			keyRV := reflect.ValueOf(key)
			if !keyRV.Type().AssignableTo(keyType) {
				return fmt.Errorf("cannot assign key %T to %v", key, keyType)
			}

			valRV := reflect.ValueOf(value)
			if !valRV.Type().AssignableTo(valueType) {
				return fmt.Errorf("cannot assign value %T to %v", value, valueType)
			}

			// 设置map中的值
			rv.SetMapIndex(keyRV, valRV)
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
	// 使用回退函数获取最新数据
	if fallback != nil {
		values, err := fallback(ctx, keys)
		if err != nil {
			return err
		}

		// 将获取到的数据存入缓存
		ttl := time.Duration(0)
		if opts != nil {
			ttl = opts.TTL
		}

		if len(values) > 0 {
			err = c.store.MSet(ctx, values, ttl)
			if err != nil {
				return err
			}

			// 更新dstMap
			// 使用反射来处理不同的map类型
			rv := reflect.ValueOf(dstMap)
			if rv.Kind() != reflect.Ptr || rv.IsNil() {
				return fmt.Errorf("dstMap must be a non-nil pointer")
			}

			rv = rv.Elem()
			if rv.Kind() != reflect.Map {
				return fmt.Errorf("dstMap must be a pointer to a map")
			}

			// 获取map的键和值类型
			mapType := rv.Type()
			keyType := mapType.Key()
			valueType := mapType.Elem()

			for key, value := range values {
				// 创建键和值的反射值
				keyRV := reflect.ValueOf(key)
				if !keyRV.Type().AssignableTo(keyType) {
					return fmt.Errorf("cannot assign key %T to %v", key, keyType)
				}

				valRV := reflect.ValueOf(value)
				if !valRV.Type().AssignableTo(valueType) {
					return fmt.Errorf("cannot assign value %T to %v", value, valueType)
				}

				// 设置map中的值
				rv.SetMapIndex(keyRV, valRV)
			}
		}
	}

	return nil
}
