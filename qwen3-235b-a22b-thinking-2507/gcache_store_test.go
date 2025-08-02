package cache

import (
	"context"
	"testing"

	"github.com/bluele/gcache"
)

func TestGCacheStore(t *testing.T) {
	cache := gcache.New(100).
		LRU().
		Build()
	defer cache.Close()

	store := NewGCacheStore(cache)

	verifyStoreImplementation(t, store)
	verifyStoreTypeHandling(t, store)
}