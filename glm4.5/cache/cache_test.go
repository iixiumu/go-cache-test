package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-cache/internal"
	"go-cache/store"
)

func TestCache_Get(t *testing.T) {
	testStore := store.NewTestStore()
	cache := NewCache(testStore)
	ctx := context.Background()

	t.Run("Get from cache hit", func(t *testing.T) {
		// 预先设置缓存
		testStore.MSet(ctx, map[string]interface{}{"key1": "value1"}, 0)

		var result string
		found, err := cache.Get(ctx, "key1", &result, nil, nil)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if !found {
			t.Fatal("Expected to find value")
		}
		if result != "value1" {
			t.Fatalf("Expected 'value1', got '%s'", result)
		}
	})

	t.Run("Get with fallback", func(t *testing.T) {
		fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
			return "fallback_value", true, nil
		}

		var result string
		found, err := cache.Get(ctx, "key2", &result, fallback, nil)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if !found {
			t.Fatal("Expected to find value")
		}
		if result != "fallback_value" {
			t.Fatalf("Expected 'fallback_value', got '%s'", result)
		}

		// 验证结果被缓存
		var cachedResult string
		cacheFound, _ := testStore.Get(ctx, "key2", &cachedResult)
		if !cacheFound || cachedResult != "fallback_value" {
			t.Fatal("Expected fallback result to be cached")
		}
	})

	t.Run("Get without fallback", func(t *testing.T) {
		var result string
		found, err := cache.Get(ctx, "nonexistent", &result, nil, nil)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if found {
			t.Fatal("Expected not to find value")
		}
	})

	t.Run("Get with fallback error", func(t *testing.T) {
		fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
			return nil, false, errors.New("fallback error")
		}

		var result string
		found, err := cache.Get(ctx, "error_key", &result, fallback, nil)
		if err == nil {
			t.Fatal("Expected error, got nil")
		}
		if found {
			t.Fatal("Expected not to find value")
		}
	})
}

func TestCache_MGet(t *testing.T) {
	testStore := store.NewTestStore()
	cache := NewCache(testStore)
	ctx := context.Background()

	// 预先设置一些缓存
	testStore.MSet(ctx, map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}, 0)

	
	t.Run("MGet with partial cache hits", func(t *testing.T) {
		fallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
			return map[string]interface{}{
				"key3": "fallback3",
				"key4": "fallback4",
			}, nil
		}

		results := make(map[string]string)
		err := cache.MGet(ctx, []string{"key1", "key2", "key3", "key4"}, &results, fallback, nil)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(results) != 4 {
			t.Fatalf("Expected 4 results, got %d", len(results))
		}
		if results["key1"] != "value1" {
			t.Fatalf("Expected 'value1', got '%s'", results["key1"])
		}
		if results["key2"] != "value2" {
			t.Fatalf("Expected 'value2', got '%s'", results["key2"])
		}
		if results["key3"] != "fallback3" {
			t.Fatalf("Expected 'fallback3', got '%s'", results["key3"])
		}
		if results["key4"] != "fallback4" {
			t.Fatalf("Expected 'fallback4', got '%s'", results["key4"])
		}
	})

	t.Run("MGet without fallback", func(t *testing.T) {
		// 清空 results map to ensure we only get cached values
		results := make(map[string]string)
		err := cache.MGet(ctx, []string{"key1", "key3"}, &results, nil, nil)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// MGet without fallback only returns cached values
		// The test store has key1 cached, but key3 is not in the store
		if len(results) != 1 {
			t.Fatalf("Expected 1 result, got %d", len(results))
		}
		if results["key1"] != "value1" {
			t.Fatalf("Expected 'value1', got '%s'", results["key1"])
		}
	})
}

func TestCache_MDelete(t *testing.T) {
	testStore := store.NewTestStore()
	cache := NewCache(testStore)
	ctx := context.Background()

	// 预先设置缓存
	testStore.MSet(ctx, map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}, 0)

	t.Run("Delete multiple keys", func(t *testing.T) {
		deleted, err := cache.MDelete(ctx, []string{"key1", "key2", "key4"})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if deleted != 2 {
			t.Fatalf("Expected to delete 2 keys, got %d", deleted)
		}

		// 验证删除结果
		var result string
		found, _ := testStore.Get(ctx, "key1", &result)
		if found {
			t.Fatal("Expected key1 to be deleted")
		}

		found, _ = testStore.Get(ctx, "key3", &result)
		if !found {
			t.Fatal("Expected key3 to still exist")
		}
	})
}

func TestCache_MRefresh(t *testing.T) {
	testStore := store.NewTestStore()
	cache := NewCache(testStore)
	ctx := context.Background()

	// 预先设置缓存
	testStore.MSet(ctx, map[string]interface{}{
		"key1": "old_value1",
		"key2": "old_value2",
	}, 0)

	t.Run("Refresh multiple keys", func(t *testing.T) {
		fallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
			return map[string]interface{}{
				"key1": "new_value1",
				"key2": "new_value2",
				"key3": "new_value3",
			}, nil
		}

		results := make(map[string]string)
		err := cache.MRefresh(ctx, []string{"key1", "key2", "key3"}, &results, fallback, nil)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(results) != 3 {
			t.Fatalf("Expected 3 results, got %d", len(results))
		}
		if results["key1"] != "new_value1" {
			t.Fatalf("Expected 'new_value1', got '%s'", results["key1"])
		}
		if results["key2"] != "new_value2" {
			t.Fatalf("Expected 'new_value2', got '%s'", results["key2"])
		}
		if results["key3"] != "new_value3" {
			t.Fatalf("Expected 'new_value3', got '%s'", results["key3"])
		}

		// 验证缓存中的值也被更新
		var cachedResult string
		found, _ := testStore.Get(ctx, "key1", &cachedResult)
		if !found || cachedResult != "new_value1" {
			t.Fatal("Expected key1 to be updated in cache")
		}
	})
}

func TestCache_WithTTL(t *testing.T) {
	testStore := store.NewTestStore()
	cache := NewCache(testStore)
	ctx := context.Background()

	t.Run("Get with TTL", func(t *testing.T) {
		fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
			return "ttl_value", true, nil
		}

		opts := &internal.CacheOptions{TTL: time.Hour}
		var result string
		found, err := cache.Get(ctx, "ttl_key", &result, fallback, opts)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if !found {
			t.Fatal("Expected to find value")
		}
		if result != "ttl_value" {
			t.Fatalf("Expected 'ttl_value', got '%s'", result)
		}
	})
}

func TestCache_WithDifferentTypes(t *testing.T) {
	testStore := store.NewTestStore()
	cache := NewCache(testStore)
	ctx := context.Background()

	t.Run("Get with different types", func(t *testing.T) {
		fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
			switch key {
			case "string_key":
				return "string_value", true, nil
			case "int_key":
				return 42, true, nil
			case "struct_key":
				return struct{ Name string }{Name: "test"}, true, nil
			default:
				return nil, false, nil
			}
		}

		// 测试字符串
		var strResult string
		found, err := cache.Get(ctx, "string_key", &strResult, fallback, nil)
		if err != nil || !found || strResult != "string_value" {
			t.Fatal("Failed to get string value")
		}

		// 测试整数
		var intResult int
		found, err = cache.Get(ctx, "int_key", &intResult, fallback, nil)
		if err != nil || !found || intResult != 42 {
			t.Fatal("Failed to get int value")
		}

		// 测试结构体
		var structResult struct{ Name string }
		found, err = cache.Get(ctx, "struct_key", &structResult, fallback, nil)
		if err != nil || !found || structResult.Name != "test" {
			t.Fatal("Failed to get struct value")
		}
	})
}