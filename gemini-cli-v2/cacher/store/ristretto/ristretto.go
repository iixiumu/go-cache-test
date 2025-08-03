package ristretto

import (
	"context"
	"errors"
	"reflect"
	"time"

	"github.com/dgraph-io/ristretto/v2"
)

// RistrettoStore is a store implementation for Ristretto.
// It stores values directly, no serialization is needed.
type RistrettoStore struct {
	cache *ristretto.Cache[string, interface{}]
}

// NewRistrettoStore creates a new RistrettoStore.
func NewRistrettoStore(cache *ristretto.Cache[string, interface{}]) *RistrettoStore {
	return &RistrettoStore{cache: cache}
}

// Get retrieves a value from ristretto.
func (s *RistrettoStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	if dst == nil {
		return false, errors.New("dst must not be nil")
	}
	if reflect.ValueOf(dst).Kind() != reflect.Ptr {
		return false, errors.New("dst must be a pointer")
	}

	val, found := s.cache.Get(key)
	if !found {
		return false, nil
	}

	dstVal := reflect.ValueOf(dst).Elem()
	srcVal := reflect.ValueOf(val)

	if srcVal.Type().AssignableTo(dstVal.Type()) {
		dstVal.Set(srcVal)
	} else if srcVal.Type().ConvertibleTo(dstVal.Type()) {
		dstVal.Set(srcVal.Convert(dstVal.Type()))
	} else if srcVal.Kind() == reflect.Ptr && srcVal.Elem().Type().AssignableTo(dstVal.Type()) {
		dstVal.Set(srcVal.Elem())
	} else {
		return false, errors.New("cannot assign value to dst")
	}

	return true, nil
}

// MGet retrieves multiple values from ristretto.
func (s *RistrettoStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	if dstMap == nil {
		return errors.New("dstMap must not be nil")
	}
	dstVal := reflect.ValueOf(dstMap)
	if dstVal.Kind() != reflect.Ptr || dstVal.Elem().Kind() != reflect.Map {
		return errors.New("dstMap must be a pointer to a map")
	}

	mapVal := dstVal.Elem()
	for _, key := range keys {
		val, found := s.cache.Get(key)
		if found {
			mapVal.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(val))
		}
	}

	return nil
}

// Exists checks if keys exist in ristretto.
func (s *RistrettoStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	existsMap := make(map[string]bool)
	for _, key := range keys {
		_, found := s.cache.Get(key)
		existsMap[key] = found
	}
	return existsMap, nil
}

// MSet sets multiple key-value pairs in ristretto.
func (s *RistrettoStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	for key, val := range items {
		s.cache.SetWithTTL(key, val, 1, ttl)
	}
	// Wait for the value to be processed by the cache.
	s.cache.Wait()
	return nil
}

// Del deletes keys from ristretto.
func (s *RistrettoStore) Del(ctx context.Context, keys ...string) (int64, error) {
	var count int64
	for _, key := range keys {
		s.cache.Del(key)
		count++
	}
	// Wait for the value to be processed by the cache.
	s.cache.Wait()
	return count, nil
}