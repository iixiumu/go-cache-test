package cache

import (
	"context"
	"testing"
	"time"
)

func TestCacher_Get(t *testing.T) {
	store := NewMockStore()
	cacher := NewCacher(store)
	ctx := context.Background()

	// 测试缓存命中
	store.(*MockStore).data["key1"] = "value1"
	var result string
	found, err := cacher.Get(ctx, "key1", &result, nil, nil)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Fatal("Expected to find key1")
	}
	if result != "value1" {
		t.Fatalf("Expected 'value1', got '%s'", result)
	}

	// 测试缓存未命中，有回退函数
	var result2 string
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		if key == "key2" {
			return "value2", true, nil
		}
		return nil, false, nil
	}

	found, err = cacher.Get(ctx, "key2", &result2, fallback, &CacheOptions{TTL: time.Hour})
	if err != nil {
		t.Fatalf("Get with fallback failed: %v", err)
	}
	if !found {
		t.Fatal("Expected to find key2 via fallback")
	}
	if result2 != "value2" {
		t.Fatalf("Expected 'value2', got '%s'", result2)
	}

	// 验证回退结果被缓存
	var cachedResult string
	found, err = cacher.Get(ctx, "key2", &cachedResult, nil, nil)
	if err != nil {
		t.Fatalf("Get cached value failed: %v", err)
	}
	if !found {
		t.Fatal("Expected to find cached key2")
	}
	if cachedResult != "value2" {
		t.Fatalf("Expected cached 'value2', got '%s'", cachedResult)
	}
}

func TestCacher_MGet(t *testing.T) {
	store := NewMockStore()
	cacher := NewCacher(store)
	ctx := context.Background()

	// 设置一些初始数据
	store.(*MockStore).data["key1"] = "value1"
	store.(*MockStore).data["key2"] = "value2"

	// 测试批量获取，部分命中
	result := make(map[string]string)
	fallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		result := make(map[string]interface{})
		for _, key := range keys {
			if key == "key3" {
				result[key] = "value3"
			}
		}
		return result, nil
	}

	err := cacher.MGet(ctx, []string{"key1", "key2", "key3"}, &result, fallback, &CacheOptions{TTL: time.Hour})
	if err != nil {
		t.Fatalf("MGet failed: %v", err)
	}

	expected := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	for key, expectedValue := range expected {
		if actualValue, exists := result[key]; !exists {
			t.Fatalf("Expected key '%s' to exist", key)
		} else if actualValue != expectedValue {
			t.Fatalf("Expected '%s' for key '%s', got '%s'", expectedValue, key, actualValue)
		}
	}
}

func TestCacher_MDelete(t *testing.T) {
	store := NewMockStore()
	cacher := NewCacher(store)
	ctx := context.Background()

	// 设置一些数据
	store.(*MockStore).data["key1"] = "value1"
	store.(*MockStore).data["key2"] = "value2"
	store.(*MockStore).data["key3"] = "value3"

	// 删除部分键
	deleted, err := cacher.MDelete(ctx, []string{"key1", "key2", "key4"})
	if err != nil {
		t.Fatalf("MDelete failed: %v", err)
	}

	if deleted != 2 {
		t.Fatalf("Expected to delete 2 keys, deleted %d", deleted)
	}

	// 验证删除结果
	if _, exists := store.(*MockStore).data["key1"]; exists {
		t.Fatal("Expected key1 to be deleted")
	}
	if _, exists := store.(*MockStore).data["key2"]; exists {
		t.Fatal("Expected key2 to be deleted")
	}
	if _, exists := store.(*MockStore).data["key3"]; !exists {
		t.Fatal("Expected key3 to still exist")
	}
}

func TestCacher_MRefresh(t *testing.T) {
	store := NewMockStore()
	cacher := NewCacher(store)
	ctx := context.Background()

	// 设置一些初始数据
	store.(*MockStore).data["key1"] = "old_value1"
	store.(*MockStore).data["key2"] = "old_value2"

	// 强制刷新
	result := make(map[string]string)
	fallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		result := make(map[string]interface{})
		for _, key := range keys {
			if key == "key1" {
				result[key] = "new_value1"
			} else if key == "key2" {
				result[key] = "new_value2"
			}
		}
		return result, nil
	}

	err := cacher.MRefresh(ctx, []string{"key1", "key2"}, &result, fallback, &CacheOptions{TTL: time.Hour})
	if err != nil {
		t.Fatalf("MRefresh failed: %v", err)
	}

	expected := map[string]string{
		"key1": "new_value1",
		"key2": "new_value2",
	}

	for key, expectedValue := range expected {
		if actualValue, exists := result[key]; !exists {
			t.Fatalf("Expected key '%s' to exist", key)
		} else if actualValue != expectedValue {
			t.Fatalf("Expected '%s' for key '%s', got '%s'", expectedValue, key, actualValue)
		}
	}

	// 验证缓存也被更新
	var cachedResult string
	found, err := cacher.Get(ctx, "key1", &cachedResult, nil, nil)
	if err != nil {
		t.Fatalf("Get refreshed value failed: %v", err)
	}
	if !found {
		t.Fatal("Expected to find refreshed key1")
	}
	if cachedResult != "new_value1" {
		t.Fatalf("Expected refreshed 'new_value1', got '%s'", cachedResult)
	}
}
