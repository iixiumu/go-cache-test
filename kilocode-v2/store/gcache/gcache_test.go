package gcache

import (
	"context"
	"testing"
	"time"
)

// TestGCacheStore_Get 测试 Get 方法
func TestGCacheStore_Get(t *testing.T) {
	// 创建 GCacheStore
	store, err := NewGCacheStore(&Config{
		Size: 1000,
	})
	if err != nil {
		t.Fatalf("Failed to create GCacheStore: %v", err)
	}
	defer store.Close()

	// 准备测试数据
	ctx := context.Background()
	key := "test_key"
	value := "test_value"

	// 先设置一个值
	err = store.MSet(ctx, map[string]interface{}{key: value}, 0)
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	// 测试获取存在的值
	var result string
	found, err := store.Get(ctx, key, &result)
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if !found {
		t.Error("Expected to find the key, but it was not found")
	}
	if result != value {
		t.Errorf("Expected %s, got %s", value, result)
	}

	// 测试获取不存在的值
	var result2 string
	found, err = store.Get(ctx, "nonexistent_key", &result2)
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if found {
		t.Error("Expected not to find the key, but it was found")
	}
}

// TestGCacheStore_MGet 测试 MGet 方法
func TestGCacheStore_MGet(t *testing.T) {
	// 创建 GCacheStore
	store, err := NewGCacheStore(&Config{
		Size: 1000,
	})
	if err != nil {
		t.Fatalf("Failed to create GCacheStore: %v", err)
	}
	defer store.Close()

	// 准备测试数据
	ctx := context.Background()
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	// 设置多个值
	err = store.MSet(ctx, items, 0)
	if err != nil {
		t.Fatalf("Failed to set values: %v", err)
	}

	// 测试批量获取
	result := make(map[string]string)
	err = store.MGet(ctx, []string{"key1", "key2", "key3", "key4"}, &result)
	if err != nil {
		t.Errorf("MGet failed: %v", err)
	}

	// 验证结果
	if len(result) != 3 {
		t.Errorf("Expected 3 items, got %d", len(result))
	}
	for k, v := range items {
		if result[k] != v {
			t.Errorf("Expected %s for key %s, got %s", v, k, result[k])
		}
	}
}

// TestGCacheStore_Exists 测试 Exists 方法
func TestGCacheStore_Exists(t *testing.T) {
	// 创建 GCacheStore
	store, err := NewGCacheStore(&Config{
		Size: 1000,
	})
	if err != nil {
		t.Fatalf("Failed to create GCacheStore: %v", err)
	}
	defer store.Close()

	// 准备测试数据
	ctx := context.Background()
	key := "test_key"
	value := "test_value"

	// 先设置一个值
	err = store.MSet(ctx, map[string]interface{}{key: value}, 0)
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}

	// 测试检查键存在性
	exists, err := store.Exists(ctx, []string{"test_key", "nonexistent_key"})
	if err != nil {
		t.Errorf("Exists failed: %v", err)
	}
	if !exists["test_key"] {
		t.Error("Expected test_key to exist")
	}
	if exists["nonexistent_key"] {
		t.Error("Expected nonexistent_key to not exist")
	}
}

// TestGCacheStore_MSet 测试 MSet 方法
func TestGCacheStore_MSet(t *testing.T) {
	// 创建 GCacheStore
	store, err := NewGCacheStore(&Config{
		Size: 1000,
	})
	if err != nil {
		t.Fatalf("Failed to create GCacheStore: %v", err)
	}
	defer store.Close()

	// 准备测试数据
	ctx := context.Background()
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	// 测试批量设置
	err = store.MSet(ctx, items, 0)
	if err != nil {
		t.Errorf("MSet failed: %v", err)
	}

	// 验证值是否正确设置
	for k, v := range items {
		var result string
		found, err := store.Get(ctx, k, &result)
		if err != nil {
			t.Errorf("Failed to get value for key %s: %v", k, err)
			continue
		}
		if !found {
			t.Errorf("Expected key %s to exist", k)
			continue
		}
		if result != v {
			t.Errorf("Expected %s for key %s, got %s", v, k, result)
		}
	}

	// 测试带 TTL 的批量设置
	err = store.MSet(ctx, items, time.Second*10)
	if err != nil {
		t.Errorf("MSet with TTL failed: %v", err)
	}
}

// TestGCacheStore_Del 测试 Del 方法
func TestGCacheStore_Del(t *testing.T) {
	// 创建 GCacheStore
	store, err := NewGCacheStore(&Config{
		Size: 1000,
	})
	if err != nil {
		t.Fatalf("Failed to create GCacheStore: %v", err)
	}
	defer store.Close()

	// 准备测试数据
	ctx := context.Background()
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	// 设置多个值
	err = store.MSet(ctx, items, 0)
	if err != nil {
		t.Fatalf("Failed to set values: %v", err)
	}

	// 测试删除
	count, err := store.Del(ctx, "key1", "key2", "nonexistent_key")
	if err != nil {
		t.Errorf("Del failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected to delete 2 keys, got %d", count)
	}

	// 验证键是否被删除
	for _, k := range []string{"key1", "key2"} {
		var result string
		found, err := store.Get(ctx, k, &result)
		if err != nil {
			t.Errorf("Failed to get value for key %s: %v", k, err)
			continue
		}
		if found {
			t.Errorf("Expected key %s to be deleted", k)
		}
	}

	// 验证未删除的键仍然存在
	var result string
	found, err := store.Get(ctx, "key3", &result)
	if err != nil {
		t.Errorf("Failed to get value for key3: %v", err)
	}
	if !found {
		t.Error("Expected key3 to exist")
	}
	if result != "value3" {
		t.Errorf("Expected value3 for key3, got %s", result)
	}
}
