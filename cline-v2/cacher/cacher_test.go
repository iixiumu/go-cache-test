package cacher

import (
	"context"
	"testing"

	"go-cache/cacher/store/ristretto"
)

func TestCacherGet(t *testing.T) {
	// 创建Ristretto存储
	store, err := ristretto.NewRistrettoStore()
	if err != nil {
		t.Fatalf("failed to create ristretto store: %v", err)
	}

	cacher := NewCacher(store)

	ctx := context.Background()

	t.Run("TestGetFromCache", func(t *testing.T) {
		key := "test_key"
		value := "test_value"

		// 预先设置缓存
		err := store.MSet(ctx, map[string]interface{}{key: value}, 0)
		if err != nil {
			t.Fatalf("MSet failed: %v", err)
		}

		var result string
		found, err := cacher.Get(ctx, key, &result, nil, nil)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if !found {
			t.Error("Key should be found")
		}

		if result != value {
			t.Errorf("Expected %v, got %v", value, result)
		}
	})

	t.Run("TestGetWithFallback", func(t *testing.T) {
		key := "fallback_key"
		expectedValue := "fallback_value"

		fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
			return expectedValue, true, nil
		}

		var result string
		found, err := cacher.Get(ctx, key, &result, fallback, nil)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if !found {
			t.Error("Key should be found through fallback")
		}

		if result != expectedValue {
			t.Errorf("Expected %v, got %v", expectedValue, result)
		}

		// 验证值已被缓存
		var cachedResult string
		found, err = store.Get(ctx, key, &cachedResult)
		if err != nil {
			t.Fatalf("Get from store failed: %v", err)
		}

		if !found {
			t.Error("Value should be cached")
		}

		if cachedResult != expectedValue {
			t.Errorf("Expected cached %v, got %v", expectedValue, cachedResult)
		}
	})

	t.Run("TestGetNotFound", func(t *testing.T) {
		key := "not_found_key"

		fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
			return nil, false, nil
		}

		var result string
		found, err := cacher.Get(ctx, key, &result, fallback, nil)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if found {
			t.Error("Key should not be found")
		}
	})
}

func TestCacherMGet(t *testing.T) {
	// 创建Ristretto存储
	store, err := ristretto.NewRistrettoStore()
	if err != nil {
		t.Fatalf("failed to create ristretto store: %v", err)
	}

	cacher := NewCacher(store)

	ctx := context.Background()

	t.Run("TestMGetPartial", func(t *testing.T) {
		// 预先设置部分缓存
		cachedItems := map[string]interface{}{
			"key1": "value1",
			"key3": "value3",
		}
		err := store.MSet(ctx, cachedItems, 0)
		if err != nil {
			t.Fatalf("MSet failed: %v", err)
		}

		keys := []string{"key1", "key2", "key3", "key4"}
		result := make(map[string]string)

		fallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
			fallbackValues := make(map[string]interface{})
			for _, key := range keys {
				fallbackValues[key] = "fallback_" + key
			}
			return fallbackValues, nil
		}

		err = cacher.MGet(ctx, keys, &result, fallback, nil)
		if err != nil {
			t.Fatalf("MGet failed: %v", err)
		}

		// 检查结果
		expected := map[string]string{
			"key1": "value1",        // 从缓存获取
			"key2": "fallback_key2", // 从回退获取
			"key3": "value3",        // 从缓存获取
			"key4": "fallback_key4", // 从回退获取
		}

		if len(result) != len(expected) {
			t.Errorf("Expected %d results, got %d", len(expected), len(result))
		}

		for k, v := range expected {
			if result[k] != v {
				t.Errorf("Key %s: expected %v, got %v", k, v, result[k])
			}
		}
	})
}

func TestCacherMDelete(t *testing.T) {
	// 创建Ristretto存储
	store, err := ristretto.NewRistrettoStore()
	if err != nil {
		t.Fatalf("failed to create ristretto store: %v", err)
	}

	cacher := NewCacher(store)

	ctx := context.Background()

	// 设置一些缓存项
	items := map[string]interface{}{
		"delete_key1": "value1",
		"delete_key2": "value2",
		"delete_key3": "value3",
	}
	err = store.MSet(ctx, items, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 删除部分键
	keys := []string{"delete_key1", "delete_key2", "non_existent_key"}
	deleted, err := cacher.MDelete(ctx, keys)
	if err != nil {
		t.Fatalf("MDelete failed: %v", err)
	}

	if deleted != 2 {
		t.Errorf("Expected 2 deletions, got %d", deleted)
	}

	// 验证删除
	var result string
	found, err := store.Get(ctx, "delete_key1", &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if found {
		t.Error("delete_key1 should be deleted")
	}
}

func TestCacherMRefresh(t *testing.T) {
	// 创建Ristretto存储
	store, err := ristretto.NewRistrettoStore()
	if err != nil {
		t.Fatalf("failed to create ristretto store: %v", err)
	}

	cacher := NewCacher(store)

	ctx := context.Background()

	// 设置初始缓存
	initialItems := map[string]interface{}{
		"refresh_key1": "old_value1",
		"refresh_key2": "old_value2",
	}
	err = store.MSet(ctx, initialItems, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 刷新键
	keys := []string{"refresh_key1", "refresh_key2", "refresh_key3"}
	result := make(map[string]string)

	fallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		fallbackValues := make(map[string]interface{})
		for _, key := range keys {
			fallbackValues[key] = "new_" + key
		}
		return fallbackValues, nil
	}

	err = cacher.MRefresh(ctx, keys, &result, fallback, nil)
	if err != nil {
		t.Fatalf("MRefresh failed: %v", err)
	}

	// 验证缓存已更新，直接从存储中获取
	finalResult := make(map[string]string)
	err = store.MGet(ctx, keys, &finalResult)
	if err != nil {
		t.Fatalf("MGet failed: %v", err)
	}

	// 检查结果
	expected := map[string]string{
		"refresh_key1": "new_refresh_key1",
		"refresh_key2": "new_refresh_key2",
		"refresh_key3": "new_refresh_key3",
	}

	for k, v := range expected {
		if finalResult[k] != v {
			t.Errorf("Key %s: expected %v, got %v", k, v, finalResult[k])
		}
	}
}
