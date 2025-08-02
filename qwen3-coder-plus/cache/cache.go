package cache

import (
	"context"
	"reflect"
	"time"
)

// cacheImpl Cacher接口的实现
type cacheImpl struct {
	store Store
}

// New 创建一个新的Cacher实例
func New(store Store) Cacher {
	return &cacheImpl{
		store: store,
	}
}

// Get 获取单个缓存项，缓存未命中时执行回退函数并缓存结果
func (c *cacheImpl) Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error) {
	// 首先尝试从缓存中获取
	found, err := c.store.Get(ctx, key, dst)
	if err != nil {
		return false, err
	}

	// 如果找到了直接返回
	if found {
		return true, nil
	}

	// 如果没有找到且没有提供回退函数，则返回false
	if fallback == nil {
		return false, nil
	}

	// 执行回退函数获取数据
	value, found, err := fallback(ctx, key)
	if err != nil || !found {
		return found, err
	}

	// 将获取到的数据存入缓存
	items := map[string]interface{}{key: value}
	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}

	err = c.store.MSet(ctx, items, ttl)
	if err != nil {
		// 如果缓存失败，我们仍然返回获取到的数据
		return true, nil
	}

	// 将值赋给dst
	reflect.ValueOf(dst).Elem().Set(reflect.ValueOf(value))

	return true, nil
}

// MGet 批量获取缓存项，支持部分命中和批量回退
func (c *cacheImpl) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	// 确保dstMap是指针类型
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr {
		return nil
	}

	// 获取map的实际值
	dstMapValue = dstMapValue.Elem()
	if dstMapValue.Kind() != reflect.Map {
		return nil
	}

	// 从缓存中批量获取
	err := c.store.MGet(ctx, keys, dstMap)
	if err != nil {
		return err
	}

	// 找出未命中的键
	missedKeys := make([]string, 0)
	hitKeys := make(map[string]bool)

	// 遍历map中的键
	for _, key := range keys {
		// 检查是否在dstMap中存在
		val := dstMapValue.MapIndex(reflect.ValueOf(key))
		if !val.IsValid() {
			missedKeys = append(missedKeys, key)
		} else {
			hitKeys[key] = true
		}
	}

	// 如果没有未命中的键，直接返回
	if len(missedKeys) == 0 {
		return nil
	}

	// 如果没有提供回退函数，直接返回
	if fallback == nil {
		return nil
	}

	// 执行批量回退函数
	fallbackValues, err := fallback(ctx, missedKeys)
	if err != nil {
		return err
	}

	// 将回退函数返回的数据存入缓存
	if len(fallbackValues) > 0 {
		ttl := time.Duration(0)
		if opts != nil {
			ttl = opts.TTL
		}

		err = c.store.MSet(ctx, fallbackValues, ttl)
		if err != nil {
			// 即使缓存失败，我们仍然继续处理
		}

		// 将回退的数据合并到结果中
		for key, value := range fallbackValues {
			dstMapValue.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
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
	// 清除现有的缓存项
	_, err := c.store.Del(ctx, keys...)
	if err != nil {
		return err
	}

	// 如果没有提供回退函数，直接返回
	if fallback == nil {
		return nil
	}

	// 执行批量回退函数获取最新数据
	fallbackValues, err := fallback(ctx, keys)
	if err != nil {
		return err
	}

	// 将获取到的数据存入缓存
	if len(fallbackValues) > 0 {
		ttl := time.Duration(0)
		if opts != nil {
			ttl = opts.TTL
		}

		err = c.store.MSet(ctx, fallbackValues, ttl)
		if err != nil {
			return err
		}

		// 确保dstMap是指针类型
		dstMapValue := reflect.ValueOf(dstMap)
		if dstMapValue.Kind() != reflect.Ptr {
			return nil
		}

		// 获取map的实际值
		dstMapValue = dstMapValue.Elem()
		if dstMapValue.Kind() == reflect.Map {
			// 将数据合并到结果中
			for key, value := range fallbackValues {
				dstMapValue.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
			}
		}
	}

	return nil
}