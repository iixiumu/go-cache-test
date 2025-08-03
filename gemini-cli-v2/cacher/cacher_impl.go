package cacher

import (
	"context"
	"errors"
	"reflect"
	"time"

	"go-cache/cacher/store"
)

// CacherImpl is the implementation of the Cacher interface.
type CacherImpl struct {
	store store.Store
}

// NewCacher creates a new Cacher.
func NewCacher(store store.Store) Cacher {
	return &CacherImpl{store: store}
}

// Get gets a value from the cache. If the value is not in the cache, it calls the fallback function to get the value and sets it to the cache.
func (c *CacherImpl) Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error) {
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

	// Set the value to dst
	dstVal := reflect.ValueOf(dst).Elem()
	srcVal := reflect.ValueOf(val)
	if srcVal.Type().AssignableTo(dstVal.Type()) {
		dstVal.Set(srcVal)
	} else if srcVal.Type().ConvertibleTo(dstVal.Type()) {
		dstVal.Set(srcVal.Convert(dstVal.Type()))
	} else if srcVal.Kind() == reflect.Ptr && srcVal.Elem().Type().AssignableTo(dstVal.Type()) {
		dstVal.Set(srcVal.Elem())
	} else {
		return false, errors.New("cannot assign fallback value to dst")
	}

	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}

	err = c.store.MSet(ctx, map[string]interface{}{key: val}, ttl)
	if err != nil {
		// If setting the cache fails, we still return the value from the fallback.
		// The error from MSet can be logged here.
	}

	return true, nil
}

// MGet gets multiple values from the cache. If some values are not in the cache, it calls the fallback function to get them and sets them to the cache.
func (c *CacherImpl) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	err := c.store.MGet(ctx, keys, dstMap)
	if err != nil {
		return err
	}

	dstVal := reflect.ValueOf(dstMap).Elem()
	missingKeys := make([]string, 0)
	for _, key := range keys {
		if !dstVal.MapIndex(reflect.ValueOf(key)).IsValid() {
			missingKeys = append(missingKeys, key)
		}
	}

	if len(missingKeys) == 0 {
		return nil
	}

	if fallback == nil {
		return nil
	}

	fallbackData, err := fallback(ctx, missingKeys)
	if err != nil {
		return err
	}

	for key, val := range fallbackData {
		dstVal.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(val))
	}

	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}

	err = c.store.MSet(ctx, fallbackData, ttl)
	if err != nil {
		// Log the error
	}

	return nil
}

// MDelete deletes multiple values from the cache.
func (c *CacherImpl) MDelete(ctx context.Context, keys []string) (int64, error) {
	return c.store.Del(ctx, keys...)
}

// MRefresh refreshes multiple values in the cache.
func (c *CacherImpl) MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	if fallback == nil {
		return nil
	}

	fallbackData, err := fallback(ctx, keys)
	if err != nil {
		return err
	}

	dstVal := reflect.ValueOf(dstMap).Elem()
	for key, val := range fallbackData {
		dstVal.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(val))
	}

	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}

	err = c.store.MSet(ctx, fallbackData, ttl)
	if err != nil {
		// Log the error
	}

	return nil
}