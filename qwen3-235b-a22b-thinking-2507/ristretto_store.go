package cache

import (
	"context"
	"reflect"
	"time"

	"github.com/dgraph-io/ristretto"
)

type RistrettoStore struct {
	cache *ristretto.Cache
}

func NewRistrettoStore(cache *ristretto.Cache) Store {
	return &RistrettoStore{cache: cache}
}

func (r *RistrettoStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	val, found := r.cache.Get(key)
	if !found {
		return false, nil
	}

	dstVal := reflect.ValueOf(dst).Elem()
	valVal := reflect.ValueOf(val)
	if dstVal.Type() != valVal.Type() {
		return false, nil
	}
	dstVal.Set(valVal)
	return true, nil
}

func (r *RistrettoStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	dstMapVal := reflect.ValueOf(dstMap).Elem()
	mapType := dstMapVal.Type()
	valueType := mapType.Elem()

	for _, key := range keys {
		val, found := r.cache.Get(key)
		if !found {
			continue
		}

		elem := reflect.New(valueType).Elem()
		valVal := reflect.ValueOf(val)
		if elem.Type() != valVal.Type() {
			continue
		}
		elem.Set(valVal)
		dstMapVal.SetMapIndex(reflect.ValueOf(key), elem)
	}
	return nil
}

func (r *RistrettoStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	existsMap := make(map[string]bool)
	for _, key := range keys {
		existsMap[key] = r.cache.Has(key)
	}
	return existsMap, nil
}

func (r *RistrettoStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	for key, value := range items {
		r.cache.Set(key, value, 1) // Cost ignored for simplicity
	}
	return nil
}

func (r *RistrettoStore) Del(ctx context.Context, keys ...string) (int64, error) {
	count := 0
	for _, key := range keys {
		if r.cache.Has(key) {
			r.cache.Del(key)
			count++
		}
	}
	return int64(count), nil
}