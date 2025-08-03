package ristretto

import (
	"testing"

	"github.com/dgraph-io/ristretto/v2"
	"go-cache/cacher/store"
)

func TestRistrettoStore(t *testing.T) {
	cache, err := ristretto.NewCache[string, interface{}](&ristretto.Config[string, interface{}]{
		NumCounters: 1e7,     // number of keys to track frequency of.
		MaxCost:     1 << 30, // maximum cost of cache.
		BufferItems: 64,      // number of keys per Get buffer.
	})
	if err != nil {
		t.Fatal(err)
	}

	ristrettoStore := NewRistrettoStore(cache)

	store.TestStore(t, ristrettoStore)
}