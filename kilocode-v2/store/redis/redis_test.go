package redis

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// TestRedisStore_Get 测试 Get 方法
func TestRedisStore_Get(t *testing.T) {
	// 启动一个 miniredis 服务器用于测试
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer s.Close()

	// 创建 Redis 客户端
	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	// 创建 RedisStore
	store := NewRedisStore(client)

	// 准备测试数据
	ctx := context.Background()
	key := "test_key"
	value := "test_value"

	// 先设置一个值（需要序列化为 JSON）
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("Failed to marshal value: %v", err)
	}
	err = client.Set(ctx, key, data, 0).Err()
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

// TestRedisStore_MGet 测试 MGet 方法
func TestRedisStore_MGet(t *testing.T) {
	// 启动一个 miniredis 服务器用于测试
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer s.Close()

	// 创建 Redis 客户端
	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	// 创建 RedisStore
	store := NewRedisStore(client)

	// 准备测试数据
	ctx := context.Background()
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	// 设置多个值（需要序列化为 JSON）
	pipe := client.TxPipeline()
	for k, v := range items {
		data, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("Failed to marshal value: %v", err)
		}
		pipe.Set(ctx, k, data, 0)
	}
	_, err = pipe.Exec(ctx)
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

// TestRedisStore_Exists 测试 Exists 方法
func TestRedisStore_Exists(t *testing.T) {
	// 启动一个 miniredis 服务器用于测试
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer s.Close()

	// 创建 Redis 客户端
	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	// 创建 RedisStore
	store := NewRedisStore(client)

	// 准备测试数据
	ctx := context.Background()
	key := "test_key"
	value := "test_value"

	// 先设置一个值（需要序列化为 JSON）
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("Failed to marshal value: %v", err)
	}
	err = client.Set(ctx, key, data, 0).Err()
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

// TestRedisStore_MSet 测试 MSet 方法
func TestRedisStore_MSet(t *testing.T) {
	// 启动一个 miniredis 服务器用于测试
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer s.Close()

	// 创建 Redis 客户端
	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	// 创建 RedisStore
	store := NewRedisStore(client)

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
		data, err := client.Get(ctx, k).Result()
		if err != nil {
			t.Errorf("Failed to get value for key %s: %v", k, err)
			continue
		}
		var result string
		err = json.Unmarshal([]byte(data), &result)
		if err != nil {
			t.Errorf("Failed to unmarshal value for key %s: %v", k, err)
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

	// 验证 TTL 是否正确设置
	ttl, err := client.TTL(ctx, "key1").Result()
	if err != nil {
		t.Errorf("Failed to get TTL: %v", err)
	}
	if ttl <= 0 {
		t.Error("Expected TTL to be greater than 0")
	}
}

// TestRedisStore_Del 测试 Del 方法
func TestRedisStore_Del(t *testing.T) {
	// 启动一个 miniredis 服务器用于测试
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer s.Close()

	// 创建 Redis 客户端
	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	// 创建 RedisStore
	store := NewRedisStore(client)

	// 准备测试数据
	ctx := context.Background()
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	// 设置多个值（需要序列化为 JSON）
	pipe := client.TxPipeline()
	for k, v := range items {
		data, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("Failed to marshal value: %v", err)
		}
		pipe.Set(ctx, k, data, 0)
	}
	_, err = pipe.Exec(ctx)
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
		_, err := client.Get(ctx, k).Result()
		if err != redis.Nil {
			t.Errorf("Expected key %s to be deleted", k)
		}
	}

	// 验证未删除的键仍然存在
	data, err := client.Get(ctx, "key3").Result()
	if err != nil {
		t.Errorf("Expected key3 to exist: %v", err)
	}
	var result string
	err = json.Unmarshal([]byte(data), &result)
	if err != nil {
		t.Errorf("Failed to unmarshal value for key3: %v", err)
	}
	if result != "value3" {
		t.Errorf("Expected value3 for key3, got %s", result)
	}
}
