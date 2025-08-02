package redis

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/example/go-cache/store"
	"github.com/go-redis/redis/v8"
)

// newTestRedisStore 创建用于测试的Redis Store实例
func newTestRedisStore(t *testing.T) (store.Store, func()) {
	// 创建一个miniredis实例用于测试
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}

	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// 创建Redis Store
	store := NewRedisStore(client)

	// 返回清理函数
	cleanup := func() {
		client.Close()
		mr.Close()
	}

	return store, cleanup
}

// TestRedisStore_Get 测试RedisStore的Get方法
func TestRedisStore_Get(t *testing.T) {
	store, cleanup := newTestRedisStore(t)
	defer cleanup()

	ctx := context.Background()

	// 测试获取不存在的键
	var result string
	found, err := store.Get(ctx, "nonexistent", &result)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}

	if found {
		t.Error("Get should not have found nonexistent key")
	}

	// 设置一个值
	testKey := "test_key"
	testValue := "test_value"
	items := map[string]interface{}{testKey: testValue}
	err = store.MSet(ctx, items, 0)
	if err != nil {
		t.Fatalf("MSet returned error: %v", err)
	}

	// 测试获取存在的键
	var result2 string
	found, err = store.Get(ctx, testKey, &result2)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}

	if !found {
		t.Error("Get should have found the key")
	}

	if result2 != testValue {
		t.Errorf("Get returned wrong value: got %v, want %v", result2, testValue)
	}
}

// TestRedisStore_MGet 测试RedisStore的MGet方法
func TestRedisStore_MGet(t *testing.T) {
	store, cleanup := newTestRedisStore(t)
	defer cleanup()

	ctx := context.Background()

	// 设置一些测试数据
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	err := store.MSet(ctx, items, 0)
	if err != nil {
		t.Fatalf("MSet returned error: %v", err)
	}

	// 测试批量获取
	keys := []string{"key1", "key2", "key3", "key4"}
	result := make(map[string]interface{})
	err = store.MGet(ctx, keys, &result)
	if err != nil {
		t.Fatalf("MGet returned error: %v", err)
	}

	// 验证结果
	expected := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	// 检查期望的键值对
	for key, expectedValue := range expected {
		if value, exists := result[key]; !exists || value != expectedValue {
			t.Errorf("MGet result mismatch for key %s: got %v, want %v", key, value, expectedValue)
		}
	}

	// 检查不存在的键
	if _, exists := result["key4"]; exists {
		t.Error("MGet should not have returned value for nonexistent key")
	}
}

// TestRedisStore_Exists 测试RedisStore的Exists方法
func TestRedisStore_Exists(t *testing.T) {
	store, cleanup := newTestRedisStore(t)
	defer cleanup()

	ctx := context.Background()

	// 设置一些测试数据
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	err := store.MSet(ctx, items, 0)
	if err != nil {
		t.Fatalf("MSet returned error: %v", err)
	}

	// 测试批量检查存在性
	keys := []string{"key1", "key2", "key3"}
	result, err := store.Exists(ctx, keys)
	if err != nil {
		t.Fatalf("Exists returned error: %v", err)
	}

	// 验证结果
	expected := map[string]bool{
		"key1": true,
		"key2": true,
		"key3": false,
	}

	for key, expectedExists := range expected {
		if exists, ok := result[key]; !ok || exists != expectedExists {
			t.Errorf("Exists result mismatch for key %s: got %v, want %v", key, exists, expectedExists)
		}
	}
}

// TestRedisStore_MSet 测试RedisStore的MSet方法
func TestRedisStore_MSet(t *testing.T) {
	store, cleanup := newTestRedisStore(t)
	defer cleanup()

	ctx := context.Background()

	// 测试批量设置
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	err := store.MSet(ctx, items, 0)
	if err != nil {
		t.Fatalf("MSet returned error: %v", err)
	}

	// 验证设置的值
	for key, expectedValue := range items {
		var result string
		found, err := store.Get(ctx, key, &result)
		if err != nil {
			t.Fatalf("Get returned error: %v", err)
		}

		if !found {
			t.Errorf("Key %s should exist", key)
		}

		if result != expectedValue {
			t.Errorf("Key %s has wrong value: got %v, want %v", key, result, expectedValue)
		}
	}
}

// TestRedisStore_Del 测试RedisStore的Del方法
func TestRedisStore_Del(t *testing.T) {
	store, cleanup := newTestRedisStore(t)
	defer cleanup()

	ctx := context.Background()

	// 设置一些测试数据
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	err := store.MSet(ctx, items, 0)
	if err != nil {
		t.Fatalf("MSet returned error: %v", err)
	}

	// 测试删除
	keys := []string{"key1", "key3", "key4"} // key4不存在
	deleted, err := store.Del(ctx, keys...)
	if err != nil {
		t.Fatalf("Del returned error: %v", err)
	}

	if deleted != 2 {
		t.Errorf("Del returned wrong count: got %d, want %d", deleted, 2)
	}

	// 验证键已被删除
	for _, key := range []string{"key1", "key3"} {
		found, err := store.Get(ctx, key, &struct{}{})
		if err != nil {
			t.Fatalf("Get returned error: %v", err)
		}

		if found {
			t.Errorf("Key %s should have been deleted", key)
		}
	}

	// 验证未删除的键仍然存在
	var result string
	found, err := store.Get(ctx, "key2", &result)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}

	if !found {
		t.Error("Key key2 should still exist")
	}

	if result != "value2" {
		t.Errorf("Key key2 has wrong value: got %v, want %v", result, "value2")
	}
}
