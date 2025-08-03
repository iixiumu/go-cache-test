package cacher

import (
	"context"
	"go-cache/store"
	"reflect"
	"time"
)

// cacherImpl 缓存实现
type cacherImpl struct {
	store store.Store
}

// NewCacher 创建Cacher实例
func NewCacher(store store.Store) Cacher {
	return &cacherImpl{store: store}
}

// Get 获取单个缓存项
func (c *cacherImpl) Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error) {
	found, err := c.store.Get(ctx, key, dst)
	if err != nil {
		return false, err
	}
	if found {
		return true, nil
	}

	if fallback == nil {
		return false, nil
	}

	val, found, err := fallback(ctx, key)
	if err != nil {
		return false, err
	}
	if !found {
		return false, nil
	}

	// set dst
	reflect.ValueOf(dst).Elem().Set(reflect.ValueOf(val))

	// set cache
	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}
	return true, c.store.MSet(ctx, map[string]interface{}{key: val}, ttl)
}

// MGet 批量获取缓存项
func (c *cacherImpl) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	err := c.store.MGet(ctx, keys, dstMap)
	if err != nil {
		return err
	}

	mapVal := reflect.ValueOf(dstMap).Elem()
	missingKeys := make([]string, 0)
	for _, key := range keys {
		if !mapVal.MapIndex(reflect.ValueOf(key)).IsValid() {
			missingKeys = append(missingKeys, key)
		}
	}

	if len(missingKeys) == 0 || fallback == nil {
		return nil
	}

	fbResult, err := fallback(ctx, missingKeys)
	if err != nil {
		return err
	}

	// set dstMap
	for key, val := range fbResult {
		mapVal.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(val))
	}

	// set cache
	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}
	return c.store.MSet(ctx, fbResult, ttl)
}

// MDelete 批量删除缓存项
func (c *cacherImpl) MDelete(ctx context.Context, keys []string) (int64, error) {
	return c.store.Del(ctx, keys...)
}

// MRefresh 批量刷新缓存项
func (c *cacherImpl) MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	if fallback == nil {
		return nil
	}

	fbResult, err := fallback(ctx, keys)
	if err != nil {
		return err
	}

	// set dstMap
	mapVal := reflect.ValueOf(dstMap).Elem()
	for key, val := range fbResult {
		mapVal.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(val))
	}

	// set cache
	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}
	return c.store.MSet(ctx, fbResult, ttl)
}