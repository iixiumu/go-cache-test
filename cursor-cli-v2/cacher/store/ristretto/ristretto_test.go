package ristretto

import (
    "testing"

    dristretto "github.com/dgraph-io/ristretto/v2"

    "go-cache/cacher/store"
    "go-cache/cacher/store/storetest"
)

func newTestRistrettoStore(t *testing.T) *RistrettoStore {
    t.Helper()
    cache, err := dristretto.NewCache[string, any](&dristretto.Config[string, any]{
        NumCounters: 1e4,
        MaxCost:     1 << 20, // 1MB
        BufferItems: 64,
    })
    if err != nil {
        t.Fatalf("failed to create ristretto: %v", err)
    }
    t.Cleanup(cache.Close)
    return NewRistrettoStore(cache)
}

func TestRistrettoStore_Compliance(t *testing.T) {
    storetest.Run(t, func(t *testing.T) store.Store { return newTestRistrettoStore(t) })
}
