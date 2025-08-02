package cache

import (
	"context"
	"reflect"
	"time"
)

type cacher struct {
	store Store
}

func NewCacher(store Store) Cacher {
	return &cacher{store: store}
}

func (c *cacher) Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error) {
	found, err := c.store.Get(ctx, key, dst)
	if err != nil {
		return false, err
	}
	if found {
		return true, nil
	}

	value, found, err := fallback(ctx, key)
	if err != nil {
		return false, err
	}
	if !found {
		return false, nil
	}

	// Use reflection to set dst
	dstVal := reflect.ValueOf(dst).Elem()
	valueVal := reflect.ValueOf(value)
	if dstVal.Type() != valueVal.Type() {
		return false, nil // Type mismatch, but let fallback handle existence
	}
	dstVal.Set(valueVal)

	// Cache the value
	ttl := time.Duration(0)
	if opts != nil && opts.TTL > 0 {
		ttl = opts.TTL
	}
	c.store.MSet(ctx, map[string]interface{}{key: value}, ttl)

	return true, nil
}

func (c *cacher) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	if err := c.store.MGet(ctx, keys, dstMap); err != nil {
		return err
	}

	// Find missing keys
	dstMapVal := reflect.ValueOf(dstMap).Elem()
	missingKeys := []string{}
	for _, key := range keys {
		if !dstMapVal.MapIndex(reflect.ValueOf(key)).IsValid() {
			missingKeys = append(missingKeys, key)
		}
	}

	if len(missingKeys) == 0 {
		return nil
	}

	results, err := fallback(ctx, missingKeys)
	if err != nil {
		return err
	}

	// Merge results into dstMap
	for key, value := range results {
		dstMapVal.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
	}

	// Cache missing results
	ttl := time.Duration(0)
	if opts != nil && opts.TTL > 0 {
		ttl = opts.TTL
	}
	c.store.MSet(ctx, results, ttl)

	return nil
}

func (c *cacher) MDelete(ctx context.Context, keys []string) (int64, error) {
	return c.store.Del(ctx, keys...)
}

func (c *cacher) MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	results, err := fallback(ctx, keys)
	if err != nil {
		return err
	}

	// Update destination map
	dstMapVal := reflect.ValueOf(dstMap).Elem()
	for key, value := range results {
		dstMapVal.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
	}

	// Cache refreshed results
	ttl := time.Duration(0)
	if opts != nil && opts.TTL > 0 {
		ttl = opts.TTL
	}
	return c.store.MSet(ctx, results, ttl)
}