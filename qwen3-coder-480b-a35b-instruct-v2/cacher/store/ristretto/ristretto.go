package ristretto

import (
	"context"
	"reflect"
	"sync"
	"time"

	"github.com/dgraph-io/ristretto/v2"
)

// RistrettoStore implements the Store interface using Ristretto
type RistrettoStore struct {
	cache *ristretto.Cache[string, interface{}]
	mu    sync.RWMutex
	ttls  map[string]time.Time
}

// NewRistrettoStore creates a new RistrettoStore instance
func NewRistrettoStore() (*RistrettoStore, error) {
	cache, err := ristretto.NewCache(&ristretto.Config[string, interface{}]{
		NumCounters: 1e7,     // number of keys to track frequency of (10M).
		MaxCost:     1 << 30, // maximum cost of cache (1GB).
		BufferItems: 64,      // number of keys per Get buffer.
	})
	if err != nil {
		return nil, err
	}

	return &RistrettoStore{
		cache: cache,
		ttls:  make(map[string]time.Time),
	}, nil
}

// Get retrieves a value from Ristretto and stores it in dst
func (r *RistrettoStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check if key has expired
	if expireTime, exists := r.ttls[key]; exists {
		if time.Now().After(expireTime) {
			// Key has expired, remove it
			r.mu.RUnlock()
			r.mu.Lock()
			r.cache.Del(key)
			delete(r.ttls, key)
			r.mu.Unlock()
			r.mu.RLock()
			return false, nil
		}
	}

	// Get value from cache
	value, found := r.cache.Get(key)
	if !found {
		return false, nil
	}

	// Copy value to dst using reflection
	return true, r.copyValue(dst, value)
}

// MGet retrieves multiple values from Ristretto and stores them in dstMap
func (r *RistrettoStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Create a map to hold the results
	resultMap := make(map[string]interface{})

	// Process each key
	for _, key := range keys {
		// Check if key has expired
		if expireTime, exists := r.ttls[key]; exists {
			if time.Now().After(expireTime) {
				// Key has expired, remove it
				r.mu.RUnlock()
				r.mu.Lock()
				r.cache.Del(key)
				delete(r.ttls, key)
				r.mu.Unlock()
				r.mu.RLock()
				continue
			}
		}

		// Get value from cache
		if value, found := r.cache.Get(key); found {
			resultMap[key] = value
		}
	}

	// Convert resultMap to the expected dstMap type using reflection
	return r.convertMap(dstMap, resultMap)
}

// Exists checks the existence of multiple keys in Ristretto
func (r *RistrettoStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]bool)

	for _, key := range keys {
		// Check if key has expired
		if expireTime, exists := r.ttls[key]; exists {
			if time.Now().After(expireTime) {
				// Key has expired
				result[key] = false
			} else {
				// Key exists and hasn't expired
				_, found := r.cache.Get(key)
				result[key] = found
			}
		} else {
			// Check if key exists in cache
			_, found := r.cache.Get(key)
			result[key] = found
		}
	}

	return result, nil
}

// MSet sets multiple key-value pairs in Ristretto with optional TTL
func (r *RistrettoStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Set expiration time if TTL is specified
	var expireTime time.Time
	if ttl > 0 {
		expireTime = time.Now().Add(ttl)
	}

	// Set each item in the cache
	for key, value := range items {
		// For in-memory cache, we can store the value directly without serialization
		r.cache.Set(key, value, 1) // cost of 1 for each item
		
		// Set expiration time if TTL is specified
		if ttl > 0 {
			r.ttls[key] = expireTime
		} else {
			// Remove any existing TTL entry
			delete(r.ttls, key)
		}
	}
	
	// Wait for all items to be processed by the cache
	r.cache.Wait()

	return nil
}

// Del deletes multiple keys from Ristretto
func (r *RistrettoStore) Del(ctx context.Context, keys ...string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var deleted int64
	for _, key := range keys {
		// Check if key exists before deleting
		if _, found := r.cache.Get(key); found {
			r.cache.Del(key)
			delete(r.ttls, key)
			deleted++
		}
	}

	return deleted, nil
}

// copyValue copies a value to dst using reflection
func (r *RistrettoStore) copyValue(dst interface{}, src interface{}) error {
	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() != reflect.Ptr {
		return nil // Not a pointer, can't modify
	}

	dstElem := dstValue.Elem()
	srcValue := reflect.ValueOf(src)

	if dstElem.Type() != srcValue.Type() {
		// Types don't match, try to convert
		srcValue = srcValue.Convert(dstElem.Type())
	}

	dstElem.Set(srcValue)
	return nil
}

// convertMap converts a map[string]interface{} to the target map type using reflection
func (r *RistrettoStore) convertMap(dstMap interface{}, srcMap map[string]interface{}) error {
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

	// Get the type of the map's value
	mapValueType := dstMapElem.Type().Elem()

	// Iterate through srcMap and convert values
	for key, value := range srcMap {
		// Create a new key value
		keyValue := reflect.ValueOf(key)

		// Convert the value to the appropriate type
		valueValue := reflect.ValueOf(value)

		// If the types don't match, try to convert
		if valueValue.Type() != mapValueType {
			// Try to convert the value
			if valueValue.Type().ConvertibleTo(mapValueType) {
				convertedValue := valueValue.Convert(mapValueType)
				dstMapElem.SetMapIndex(keyValue, convertedValue)
			}
		} else {
			dstMapElem.SetMapIndex(keyValue, valueValue)
		}
	}

	return nil
}