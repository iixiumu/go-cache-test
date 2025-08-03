package ristretto

import (
	"context"
	"testing"
	"time"

	"github.com/dgraph-io/ristretto"
	"go-cache/store"
)

func newTestRistrettoStore(t *testing.T) store.Store {
	config := &ristretto.Config{
		NumCounters: 1e7,     // number of keys to track frequency of.
		MaxCost:     1 << 30, // maximum cost of cache.
		BufferItems: 64,      // number of keys per Get buffer.
	}
	cache, err := ristretto.NewCache(config)
	if err != nil {
		t.Fatalf("failed to create ristretto cache: %v", err)
	}
	return NewRistrettoStore(cache)
}

func TestRistrettoStore_Get(t *testing.T) {
	store := newTestRistrettoStore(t)
	ctx := context.Background()
	key := "test_key"
	value := "test_value"

	// Test when key exists
	store.MSet(ctx, map[string]interface{}{
		key: value,
	}, 0)

	var dst string
	found, err := store.Get(ctx, key, &dst)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !found {
		t.Errorf("Expected to find key '%s'", key)
	}
	if dst != value {
		t.Errorf("Expected value '%s', got '%s'", value, dst)
	}

	// Test when key does not exist
	store.Del(ctx, key)
	var dst2 string
	found, err = store.Get(ctx, "non_existent_key", &dst2)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if found {
		t.Errorf("Expected not to find key 'non_existent_key'")
	}
}

func TestRistrettoStore_MGet(t *testing.T) {
	store := newTestRistrettoStore(t)
	ctx := context.Background()
	keys := []string{"key1", "key2", "key3"}
	expected := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	store.MSet(ctx, map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}, 0)

	dstMap := make(map[string]string)
	err := store.MGet(ctx, keys, &dstMap)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(dstMap) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(dstMap))
	}

	for k, v := range expected {
		if dstMap[k] != v {
			t.Errorf("Expected value '%s' for key '%s', got '%s'", v, k, dstMap[k])
		}
	}
}

func TestRistrettoStore_Exists(t *testing.T) {
	store := newTestRistrettoStore(t)
	ctx := context.Background()
	keys := []string{"key1", "key2", "key3"}

	store.MSet(ctx, map[string]interface{}{
		"key1": "value1",
		"key3": "value3",
	}, 0)

	exists, err := store.Exists(ctx, keys)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !exists["key1"] {
		t.Errorf("Expected key1 to exist")
	}
	if exists["key2"] {
		t.Errorf("Expected key2 not to exist")
	}
	if !exists["key3"] {
		t.Errorf("Expected key3 to exist")
	}
}

func TestRistrettoStore_MSet(t *testing.T) {
	store := newTestRistrettoStore(t)
	ctx := context.Background()
	items := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	}
	ttl := 100 * time.Millisecond

	err := store.MSet(ctx, items, ttl)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	var val1 string
	store.Get(ctx, "key1", &val1)
	if val1 != "value1" {
		t.Errorf("Expected value for key1 to be 'value1', got '%s'", val1)
	}

	var val2 int
	store.Get(ctx, "key2", &val2)
	if val2 != 123 {
		t.Errorf("Expected value for key2 to be '123', got '%d'", val2)
	}

	time.Sleep(200 * time.Millisecond)

	var val3 string
	found, _ := store.Get(ctx, "key1", &val3)
	if found {
		t.Errorf("Expected key1 to have expired")
	}
}

func TestRistrettoStore_Del(t *testing.T) {
	store := newTestRistrettoStore(t)
	ctx := context.Background()
	keys := []string{"key1", "key2", "key3"}

	store.MSet(ctx, map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}, 0)

	deleted, err := store.Del(ctx, keys...)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if deleted != 2 {
		t.Errorf("Expected to delete 2 keys, got %d", deleted)
	}

	var val string
	found, _ := store.Get(ctx, "key1", &val)
	if found {
		t.Errorf("Expected key1 to be deleted")
	}
	found, _ = store.Get(ctx, "key2", &val)
	if found {
		t.Errorf("Expected key2 to be deleted")
	}
}