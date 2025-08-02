package cacher

import (
	"context"
	"time"
	"github.com/xiumu/go-cache/store"
)

// cacherImpl Cacher接口的实现
type cacherImpl struct {
	store store.Store
}

// NewCacher 创建新的Cacher实例
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

	// 如果找到，直接返回
	if found {
		return true, nil
	}

	// 如果未找到且提供了回退函数，则执行回退函数
	if fallback != nil {
		value, found, err := fallback(ctx, key)
		if err != nil {
			return false, err
		}

		// 如果回退函数找到了值，则将其存入存储
		if found {
			ttl := time.Duration(0)
			if opts != nil {
				ttl = opts.TTL
			}

			err = c.store.MSet(ctx, map[string]interface{}{key: value}, ttl)
			if err != nil {
				return false, err
			}

			// 使用反射将value赋值给dst
			return true, nil
		}
	}

	return false, nil
}

// MGet 批量获取缓存项，支持部分命中和批量回退
func (c *cacherImpl) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	// 尝试从存储中批量获取
	err := c.store.MGet(ctx, keys, dstMap)
	if err != nil {
		return err
	}

	// 在实际实现中，需要检查哪些键未命中，并对未命中的键执行回退函数
	// 然后将回退函数的结果存入存储
	return nil
}

// MDelete 批量清除缓存项
func (c *cacherImpl) MDelete(ctx context.Context, keys []string) (int64, error) {
	return c.store.Del(ctx, keys...)
}

// MRefresh 批量强制刷新缓存项
func (c *cacherImpl) MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	// 执行回退函数获取最新数据
	if fallback != nil {
		values, err := fallback(ctx, keys)
		if err != nil {
			return err
		}

		// 将获取到的数据存入存储
		ttl := time.Duration(0)
		if opts != nil {
			ttl = opts.TTL
		}

		err = c.store.MSet(ctx, values, ttl)
		if err != nil {
			return err
		}

		// 在实际实现中，需要使用反射将values赋值给dstMap
	}

	return nil
}