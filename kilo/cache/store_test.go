package cache

import (
	"context"
	"testing"
)

func TestStoreInterface(t *testing.T) {
	// 测试所有Store实现是否都实现了Store接口
	var _ Store = &redisStore{}
	var _ Store = &ristrettoStore{}
	var _ Store = &gcacheStore{}
	var _ Store = &memoryStore{}
}

func TestMemoryStore_Get(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// 测试获取不存在的键
	var result string
	found, err := store.Get(ctx, "nonexistent", &result)
	if err != nil {
		t.Errorf("Get returned error: %v", err)
	}
	if found {
		t.Error("Get should not have found the key")
	}

	// 测试获取存在的键
	key := "test_key"
	value := "test_value"
	store.MSet(ctx, map[string]interface{}{key: value}, 0)

	found, err = store.Get(ctx, key, &result)
	if err != nil {
		t.Errorf("Get returned error: %v", err)
	}
	if !found {
		t.Error("Get should have found the key")
	}
	if result != value {
		t.Errorf("Get returned wrong value: got %v, want %v", result, value)
	}
}

func TestMemoryStore_MGet(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// 设置测试数据
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	store.MSet(ctx, items, 0)

	// 测试批量获取
	keys := []string{"key1", "key2", "key3"}
	result := make(map[string]string)
	err := store.MGet(ctx, keys, &result)
	if err != nil {
		t.Errorf("MGet returned error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("MGet returned wrong number of items: got %v, want %v", len(result), 2)
	}

	if result["key1"] != "value1" {
		t.Errorf("MGet returned wrong value for key1: got %v, want %v", result["key1"], "value1")
	}

	if result["key2"] != "value2" {
		t.Errorf("MGet returned wrong value for key2: got %v, want %v", result["key2"], "value2")
	}
}

func TestMemoryStore_Exists(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// 设置测试数据
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	store.MSet(ctx, items, 0)

	// 测试批量检查存在性
	keys := []string{"key1", "key2", "key3"}
	result, err := store.Exists(ctx, keys)
	if err != nil {
		t.Errorf("Exists returned error: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Exists returned wrong number of items: got %v, want %v", len(result), 3)
	}

	if !result["key1"] {
		t.Error("Exists should have found key1")
	}

	if !result["key2"] {
		t.Error("Exists should have found key2")
	}

	if result["key3"] {
		t.Error("Exists should not have found key3")
	}
}

func TestMemoryStore_MSet(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// 测试批量设置
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	err := store.MSet(ctx, items, 0)
	if err != nil {
		t.Errorf("MSet returned error: %v", err)
	}

	// 验证值已设置
	var result string
	found, err := store.Get(ctx, "key1", &result)
	if err != nil {
		t.Errorf("Get returned error: %v", err)
	}
	if !found {
		t.Error("Get should have found key1")
	}
	if result != "value1" {
		t.Errorf("Get returned wrong value: got %v, want %v", result, "value1")
	}
}

func TestMemoryStore_Del(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// 设置测试数据
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	store.MSet(ctx, items, 0)

	// 测试删除
	keys := []string{"key1", "key2"}
	deleted, err := store.Del(ctx, keys...)
	if err != nil {
		t.Errorf("Del returned error: %v", err)
	}
	if deleted != 2 {
		t.Errorf("Del deleted wrong number of items: got %v, want %v", deleted, 2)
	}

	// 验证键已被删除
	for _, key := range keys {
		found, err := store.Get(ctx, key, new(string))
		if err != nil {
			t.Errorf("Get returned error: %v", err)
		}
		if found {
			t.Errorf("Key %v should have been deleted", key)
		}
	}

	// 验证未删除的键仍然存在
	found, err := store.Get(ctx, "key3", new(string))
	if err != nil {
		t.Errorf("Get returned error: %v", err)
	}
	if !found {
		t.Error("Key key3 should still exist")
	}
}
