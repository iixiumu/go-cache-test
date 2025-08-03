package cacher

import (
	"context"
	"reflect"
	"time"

	"go-cache/cacher/store"
)

// cacherImpl implements the Cacher interface
type cacherImpl struct {
	store store.Store
}

// NewCacher creates a new Cacher instance with the specified store
func NewCacher(store store.Store) Cacher {
	return &cacherImpl{
		store: store,
	}
}

// Get retrieves a value from cache, using fallback function if not found
func (c *cacherImpl) Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error) {
	// Try to get from cache first
	found, err := c.store.Get(ctx, key, dst)
	if err != nil {
		return false, err
	}

	// If found in cache, return it
	if found {
		return true, nil
	}

	// If not found and no fallback, return false
	if fallback == nil {
		return false, nil
	}

	// Execute fallback function to get the value
	value, found, err := fallback(ctx, key)
	if err != nil {
		return false, err
	}

	// If fallback didn't find the value, return false
	if !found {
		return false, nil
	}

	// Store the value in cache
	items := map[string]interface{}{key: value}
	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}

	err = c.store.MSet(ctx, items, ttl)
	if err != nil {
		// If we can't cache the value, still return it
		// Copy value to dst using reflection
		err = c.copyValue(dst, value)
		if err != nil {
			return true, err
		}
		return true, nil
	}

	// Copy value to dst using reflection
	err = c.copyValue(dst, value)
	if err != nil {
		return true, err
	}

	return true, nil
}

// MGet retrieves multiple values from cache, using fallback function for missing keys
func (c *cacherImpl) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	// Get the reflect.Value of dstMap
	dstMapValue := reflect.ValueOf(dstMap)

	// Check if dstMap is a pointer
	if dstMapValue.Kind() != reflect.Ptr {
		return nil // Not a pointer, can't modify
	}

	// Get the element that dstMap points to
	dstMapElem := dstMapValue.Elem()

	// Check if it's a map
	if dstMapElem.Kind() != reflect.Map {
		return nil // Not a map, can't convert
	}

	// Check if the map is nil and initialize it if needed
	if dstMapElem.IsNil() {
		mapType := dstMapElem.Type()
		newMap := reflect.MakeMap(mapType)
		dstMapElem.Set(newMap)
	}

	// Try to get all keys from cache
	err := c.store.MGet(ctx, keys, dstMap)
	if err != nil {
		return err
	}

	// Check which keys are missing
	missingKeys := make([]string, 0)
	dstMapInterface := dstMapElem.Interface()
	dstMapReflect := reflect.ValueOf(dstMapInterface)
	
	// Convert dstMap to map[string]interface{} for easier handling
	resultMap := make(map[string]interface{})
	if dstMapReflect.Kind() == reflect.Map {
		for _, key := range keys {
			mapValue := dstMapReflect.MapIndex(reflect.ValueOf(key))
			if !mapValue.IsValid() {
				missingKeys = append(missingKeys, key)
			} else {
				resultMap[key] = mapValue.Interface()
			}
		}
	}

	// If no fallback or no missing keys, return what we have
	if fallback == nil || len(missingKeys) == 0 {
		return nil
	}

	// Execute fallback function to get missing values
	fallbackValues, err := fallback(ctx, missingKeys)
	if err != nil {
		return err
	}

	// Add fallback values to result map
	for key, value := range fallbackValues {
		resultMap[key] = value
	}

	// Store fallback values in cache
	if len(fallbackValues) > 0 {
		ttl := time.Duration(0)
		if opts != nil {
			ttl = opts.TTL
		}

		err = c.store.MSet(ctx, fallbackValues, ttl)
		if err != nil {
			// If we can't cache the values, still return them
			// Update the dstMap with fallback values
			c.updateMap(dstMap, resultMap)
			return nil
		}
	}

	// Update the dstMap with all values
	c.updateMap(dstMap, resultMap)

	return nil
}

// MDelete deletes multiple keys from cache
func (c *cacherImpl) MDelete(ctx context.Context, keys []string) (int64, error) {
	return c.store.Del(ctx, keys...)
}

// MRefresh forces refresh of multiple keys in cache using fallback function
func (c *cacherImpl) MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	// If no fallback, nothing to refresh
	if fallback == nil {
		return nil
	}

	// Execute fallback function to get values
	values, err := fallback(ctx, keys)
	if err != nil {
		return err
	}

	// Store values in cache
	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}

	err = c.store.MSet(ctx, values, ttl)
	if err != nil {
		return err
	}

	// Update dstMap with values
	c.updateMap(dstMap, values)

	return nil
}

// copyValue copies a value to dst using reflection
func (c *cacherImpl) copyValue(dst interface{}, src interface{}) error {
	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() != reflect.Ptr {
		return nil // Not a pointer, can't modify
	}

	dstElem := dstValue.Elem()
	srcValue := reflect.ValueOf(src)

	// If types don't match, try to convert
	if dstElem.Type() != srcValue.Type() {
		if srcValue.Type().ConvertibleTo(dstElem.Type()) {
			convertedValue := srcValue.Convert(dstElem.Type())
			dstElem.Set(convertedValue)
		} else {
			// Try to use the value directly
			dstElem.Set(srcValue)
		}
	} else {
		dstElem.Set(srcValue)
	}

	return nil
}

// updateMap updates a map with values using reflection
func (c *cacherImpl) updateMap(dstMap interface{}, values map[string]interface{}) error {
	// Get the reflect.Value of dstMap
	dstMapValue := reflect.ValueOf(dstMap)

	// Check if dstMap is a pointer
	if dstMapValue.Kind() != reflect.Ptr {
		return nil // Not a pointer, can't modify
	}

	// Get the element that dstMap points to
	dstMapElem := dstMapValue.Elem()

	// Check if it's a map
	if dstMapElem.Kind() != reflect.Map {
		return nil // Not a map, can't convert
	}

	// Update the map with values
	for key, value := range values {
		keyValue := reflect.ValueOf(key)
		valueValue := reflect.ValueOf(value)
		dstMapElem.SetMapIndex(keyValue, valueValue)
	}

	return nil
}