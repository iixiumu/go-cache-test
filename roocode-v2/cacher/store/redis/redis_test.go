package redis

import (
	"context"
	"testing"
	"time"

	"go-cache/cacher/store"

	miniredis "github.com/alicebob/miniredis/v2"
	redis "github.com/redis/go-redis/v9"
)

// TestRedisStore 测试RedisStore的实现
func TestRedisStore(t *testing.T) {
	// 创建一个miniredis实例用于测试
	mr := miniredis.RunT(t)

	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// 创建RedisStore
	s := NewRedisStore(client)

	// 运行所有测试
	testGetSet(t, s)
	testMGet(t, s)
	testExists(t, s)
	testMSet(t, s)
	testDel(t, s)
	testTTL(t, s, mr)
}

func testGetSet(t *testing.T, s store.Store) {
	ctx := context.Background()

	// 测试设置和获取字符串
	key := "test_key"
	value := "test_value"

	// 设置值
	items := map[string]interface{}{key: value}
	err := s.MSet(ctx, items, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 获取值
	var result string
	found, err := s.Get(ctx, key, &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Fatalf("Key not found")
	}
	if result != value {
		t.Fatalf("Expected %s, got %s", value, result)
	}

	// 测试不存在的键
	var result2 string
	found, err = s.Get(ctx, "nonexistent", &result2)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if found {
		t.Fatalf("Expected key not found")
	}
}

func testMGet(t *testing.T, s store.Store) {
	ctx := context.Background()

	// 设置多个值
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	err := s.MSet(ctx, items, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 批量获取
	keys := []string{"key1", "key2", "key3", "nonexistent"}
	result := make(map[string]string)
	err = s.MGet(ctx, keys, &result)
	if err != nil {
		t.Fatalf("MGet failed: %v", err)
	}

	// 验证结果
	if len(result) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(result))
	}
	if result["key1"] != "value1" {
		t.Fatalf("Expected value1, got %s", result["key1"])
	}
	if result["key2"] != "value2" {
		t.Fatalf("Expected value2, got %s", result["key2"])
	}
	if result["key3"] != "value3" {
		t.Fatalf("Expected value3, got %s", result["key3"])
	}
}

func testExists(t *testing.T, s store.Store) {
	ctx := context.Background()

	// 设置值
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	err := s.MSet(ctx, items, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 检查存在性
	keys := []string{"key1", "key2", "nonexistent"}
	exists, err := s.Exists(ctx, keys)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}

	// 验证结果
	if len(exists) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(exists))
	}
	if !exists["key1"] {
		t.Fatalf("Expected key1 to exist")
	}
	if !exists["key2"] {
		t.Fatalf("Expected key2 to exist")
	}
	if exists["nonexistent"] {
		t.Fatalf("Expected nonexistent key to not exist")
	}
}

func testMSet(t *testing.T, s store.Store) {
	ctx := context.Background()

	// 批量设置
	items := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}
	err := s.MSet(ctx, items, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 验证设置的值
	var result1 string
	found, err := s.Get(ctx, "key1", &result1)
	if err != nil || !found || result1 != "value1" {
		t.Fatalf("Failed to get key1")
	}

	var result2 int
	found, err = s.Get(ctx, "key2", &result2)
	if err != nil || !found || result2 != 42 {
		t.Fatalf("Failed to get key2")
	}

	var result3 bool
	found, err = s.Get(ctx, "key3", &result3)
	if err != nil || !found || !result3 {
		t.Fatalf("Failed to get key3")
	}
}

func testDel(t *testing.T, s store.Store) {
	ctx := context.Background()

	// 设置值
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	err := s.MSet(ctx, items, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 删除部分键
	keys := []string{"key1", "key2", "nonexistent"}
	deleted, err := s.Del(ctx, keys...)
	if err != nil {
		t.Fatalf("Del failed: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("Expected 2 deleted keys, got %d", deleted)
	}

	// 验证键已被删除
	var result string
	found, err := s.Get(ctx, "key1", &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if found {
		t.Fatalf("Expected key1 to be deleted")
	}

	found, err = s.Get(ctx, "key2", &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if found {
		t.Fatalf("Expected key2 to be deleted")
	}

	// 验证未删除的键仍然存在
	found, err = s.Get(ctx, "key3", &result)
	if err != nil || !found || result != "value3" {
		t.Fatalf("Expected key3 to still exist")
	}
}

func testTTL(t *testing.T, s store.Store, mr *miniredis.Miniredis) {
	ctx := context.Background()

	// 设置带TTL的值
	key := "ttl_key"
	value := "ttl_value"
	items := map[string]interface{}{key: value}
	err := s.MSet(ctx, items, time.Second)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 验证值存在
	var result string
	found, err := s.Get(ctx, key, &result)
	if err != nil || !found || result != value {
		t.Fatalf("Failed to get ttl_key")
	}

	// 等待过期
	mr.FastForward(time.Second * 2)

	// 验证值已过期
	found, err = s.Get(ctx, key, &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if found {
		t.Fatalf("Expected ttl_key to be expired")
	}
}

// TestRedisStoreSpecific 测试RedisStore的特定功能
func TestRedisStoreSpecific(t *testing.T) {
	// 创建一个miniredis实例用于测试
	mr := miniredis.RunT(t)

	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// 创建RedisStore
	store := NewRedisStore(client)
	ctx := context.Background()

	// 测试不同数据类型的存储和获取
	t.Run("TestDifferentTypes", func(t *testing.T) {
		// 存储字符串
		items := map[string]interface{}{
			"string_key": "string_value",
			"int_key":    42,
			"bool_key":   true,
			"float_key":  3.14,
		}

		err := store.MSet(ctx, items, 0)
		if err != nil {
			t.Fatalf("MSet failed: %v", err)
		}

		// 获取字符串
		var stringValue string
		found, err := store.Get(ctx, "string_key", &stringValue)
		if err != nil || !found || stringValue != "string_value" {
			t.Fatalf("Failed to get string_key")
		}

		// 获取整数
		var intValue int
		found, err = store.Get(ctx, "int_key", &intValue)
		if err != nil || !found || intValue != 42 {
			t.Fatalf("Failed to get int_key")
		}

		// 获取布尔值
		var boolValue bool
		found, err = store.Get(ctx, "bool_key", &boolValue)
		if err != nil || !found || !boolValue {
			t.Fatalf("Failed to get bool_key")
		}

		// 获取浮点数
		var floatValue float64
		found, err = store.Get(ctx, "float_key", &floatValue)
		if err != nil || !found || floatValue != 3.14 {
			t.Fatalf("Failed to get float_key")
		}
	})

	// 测试TTL功能
	t.Run("TestTTL", func(t *testing.T) {
		// 设置带TTL的值
		items := map[string]interface{}{
			"ttl_key": "ttl_value",
		}

		err := store.MSet(ctx, items, time.Second)
		if err != nil {
			t.Fatalf("MSet failed: %v", err)
		}

		// 验证值存在
		var value string
		found, err := store.Get(ctx, "ttl_key", &value)
		if err != nil || !found || value != "ttl_value" {
			t.Fatalf("Failed to get ttl_key")
		}

		// 快进时间
		mr.FastForward(time.Second * 2)

		// 验证值已过期
		found, err = store.Get(ctx, "ttl_key", &value)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if found {
			t.Fatalf("Expected ttl_key to be expired")
		}
	})
}
