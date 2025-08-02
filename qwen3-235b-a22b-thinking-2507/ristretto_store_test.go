package cache

import (
	"context"
	"testing"
	"time"

	"github.com/dgraph-io/ristretto"
)

func TestRistrettoStore(t *testing.T) {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,
		MaxCost:     1e9,
		BufferItems: 64,
	})
	if err != nil {
		t.Fatalf("Failed to create Ristretto cache: %v", err)
	}
	defer cache.Close()

	store := NewRistrettoStore(cache)

	verifyStoreImplementation(t, store)
	verifyStoreTypeHandling(t, store)
}