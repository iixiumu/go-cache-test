package cacher

import (
	"context"
	"errors"
	"go-cache/store/gcache"
	"testing"

	bluelegcache "github.com/bluele/gcache"
)

func newTestCacher() Cacher {
	gc := bluelegcache.New(20).LRU().Build()
	store := gcache.NewGcacheStore(gc)
	return NewCacher(store)
}

func TestCacher_Get(t *testing.T) {
	cacher := newTestCacher()
	ctx := context.Background()
	key := "test_key"
	value := "test_value"

	// Test cache miss with fallback
	var dst string
	found, err := cacher.Get(ctx, key, &dst, func(ctx context.Context, key string) (interface{}, bool, error) {
		return value, true, nil
	}, nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !found {
		t.Errorf("Expected to find key '%s'", key)
	}
	if dst != value {
		t.Errorf("Expected value '%s', got '%s'", value, dst)
	}

	// Test cache hit
	var dst2 string
	found, err = cacher.Get(ctx, key, &dst2, func(ctx context.Context, key string) (interface{}, bool, error) {
		return "new_value", true, nil
	}, nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !found {
		t.Errorf("Expected to find key '%s'", key)
	}
	if dst2 != value {
		t.Errorf("Expected value from cache '%s', got '%s'", value, dst2)
	}

	// Test fallback returns not found
	var dst3 string
	found, err = cacher.Get(ctx, "not_found_key", &dst3, func(ctx context.Context, key string) (interface{}, bool, error) {
		return nil, false, nil
	}, nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if found {
		t.Errorf("Expected not to find key 'not_found_key'")
	}

	// Test fallback returns error
	var dst4 string
	_, err = cacher.Get(ctx, "error_key", &dst4, func(ctx context.Context, key string) (interface{}, bool, error) {
		return nil, false, errors.New("fallback error")
	}, nil)
	if err == nil {
		t.Errorf("Expected an error from fallback")
	}
}

func TestCacher_MGet(t *testing.T) {
	cacher := newTestCacher()
	ctx := context.Background()
	keys := []string{"key1", "key2", "key3"}

	// Pre-populate cache for key1
	cacher.Get(ctx, "key1", new(string), func(ctx context.Context, key string) (interface{}, bool, error) {
		return "value1", true, nil
	}, nil)

	dstMap := make(map[string]string)
	err := cacher.MGet(ctx, keys, &dstMap, func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		return map[string]interface{}{
			"key2": "value2",
		}, nil
	}, nil)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(dstMap) != 2 {
		t.Errorf("Expected 2 items, got %d", len(dstMap))
	}

	if dstMap["key1"] != "value1" {
		t.Errorf("Expected value1 for key1")
	}

	if dstMap["key2"] != "value2" {
		t.Errorf("Expected value2 for key2")
	}
}

func TestCacher_MDelete(t *testing.T) {
	cacher := newTestCacher()
	ctx := context.Background()
	keys := []string{"key1", "key2", "key3"}

	cacher.MGet(ctx, keys, &map[string]string{}, func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		return map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		}, nil
	}, nil)

	deleted, err := cacher.MDelete(ctx, []string{"key1", "key3"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if deleted != 1 {
		t.Errorf("Expected to delete 1 key, got %d", deleted)
	}
}

func TestCacher_MRefresh(t *testing.T) {
	cacher := newTestCacher()
	ctx := context.Background()
	keys := []string{"key1", "key2"}

	// Pre-populate cache
	cacher.MGet(ctx, keys, &map[string]string{}, func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		return map[string]interface{}{
			"key1": "old1",
			"key2": "old2",
		}, nil
	}, nil)

	dstMap := make(map[string]string)
	err := cacher.MRefresh(ctx, keys, &dstMap, func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		return map[string]interface{}{
			"key1": "new1",
			"key2": "new2",
		}, nil
	}, nil)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if dstMap["key1"] != "new1" || dstMap["key2"] != "new2" {
		t.Errorf("Refresh failed, got: %v", dstMap)
	}

	// Verify cache is updated
	var val string
	cacher.Get(ctx, "key1", &val, nil, nil)
	if val != "new1" {
		t.Errorf("Expected new1 in cache, got %s", val)
	}
}