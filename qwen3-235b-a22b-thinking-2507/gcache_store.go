package cache

import (
	"context"
	"reflect"
	"time"

	"github.com/bluele/gcache"
)

type GCacheStore struct {
	cache gcache.Cache
}

func NewGCacheStore(cache gcache.Cache) Store {
	return &GCacheStore{cache: cache}
}

func (g *GCacheStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	val, err := g.cache.Get(key)
	if err != nil {
		return false, err
	}
	if val == nil {
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

func (g *GCacheStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	dstMapVal := reflect.ValueOf(dstMap).Elem()
	mapType := dstMapVal.Type()
	valueType := mapType.Elem()

	for _, key := range keys {
		val, err := g.cache.Get(key)
		if err != nil || val == nil {
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

func (g *GCacheStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	existsMap := make(map[string]bool)
	for _, key := range keys {
		existsMap[key] = g.cache.Has(key)
	}
	return existsMap, nil
}

func (g *GCacheStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	for key, value := range items {
		if ttl > 0 {
			g.cache.SetWithExpire(key, value, ttl)
		} else {
			g.cache.Set(key, value)
		}
	}
	return nil
}

func (g *GCacheStore) Del(ctx context.Context, keys ...string) (int64, error) {
	count := 0
	for _, key := range keys {
		if g.cache.Has(key) {
			g.cache.Remove(key)
			count++
		}
	}
	return int64(count), nil
}