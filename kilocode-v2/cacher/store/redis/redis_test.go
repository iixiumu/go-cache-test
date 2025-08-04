package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisStore(t *testing.T) {
	// 创建一个miniredis实例用于测试
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// 创建RedisStore实例
	store := NewRedisStore(client)

	// 测试MSet和Get
	ctx := context.Background()
	items := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}

	err = store.MSet(ctx, items, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 测试Get
	var result string
	found, err := store.Get(ctx, "key1", &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if !found {
		t.Error("Key should be found")
	}

	if result != "value1" {
		t.Errorf("Expected value1, got %v", result)
	}

	// 测试Get with int
	var intResult int
	found, err = store.Get(ctx, "key2", &intResult)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if !found {
		t.Error("Key should be found")
	}

	if intResult != 42 {
		t.Errorf("Expected 42, got %v", intResult)
	}

	// 测试MGet
	keys := []string{"key1", "key2", "nonexistent"}
	resultMap := make(map[string]interface{})
	err = store.MGet(ctx, keys, &resultMap)
	if err != nil {
		t.Fatalf("MGet failed: %v", err)
	}

	if len(resultMap) != 2 {
		t.Errorf("Expected 2 items, got %d", len(resultMap))
	}

	// 测试Exists
	exists, err := store.Exists(ctx, keys)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}

	if len(exists) != 3 {
		t.Errorf("Expected 3 results, got %d", len(exists))
	}

	if !exists["key1"] {
		t.Error("key1 should exist")
	}

	if !exists["key2"] {
		t.Error("key2 should exist")
	}

	if exists["nonexistent"] {
		t.Error("nonexistent should not exist")
	}

	// 测试Del
	deleted, err := store.Del(ctx, "key1", "key2")
	if err != nil {
		t.Fatalf("Del failed: %v", err)
	}

	if deleted != 2 {
		t.Errorf("Expected 2 deleted keys, got %d", deleted)
	}

	// 验证键是否被删除
	found, err = store.Get(ctx, "key1", &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if found {
		t.Error("key1 should not be found after deletion")
	}
}

func TestRedisStoreWithTTL(t *testing.T) {
	// 创建一个miniredis实例用于测试
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// 创建RedisStore实例
	store := NewRedisStore(client)

	// 测试MSet with TTL
	ctx := context.Background()
	items := map[string]interface{}{
		"ttl_key": "value",
	}

	// 使用1秒作为TTL，因为这是Redis支持的最小值
	ttl := 1 * time.Second
	err = store.MSet(ctx, items, ttl)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 立即获取数据，应该能获取到
	var result string
	found, err := store.Get(ctx, "ttl_key", &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if !found {
		t.Error("ttl_key should be found immediately")
	}

	if result != "value" {
		t.Errorf("Expected value, got %v", result)
	}

	// 等待TTL过期
	time.Sleep(ttl + 100*time.Millisecond)

	// 再次获取数据，应该获取不到
	found, err = store.Get(ctx, "ttl_key", &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if found {
		t.Error("ttl_key should not be found after TTL expired")
	}
}
