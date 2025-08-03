package redis

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"go-cache/store"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

func TestRedisStore_Get(t *testing.T) {
	// 启动miniredis服务器
	s := miniredis.RunT(t)
	defer s.Close()

	// 创建redis客户端
	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	// 创建RedisStore
	rs := NewRedisStore(client)

	// 准备测试数据
	ctx := context.Background()
	testKey := "test_key"
	testValue := "test_value"

	// 序列化值
	data, err := json.Marshal(testValue)
	if err != nil {
		t.Fatalf("Failed to marshal test value: %v", err)
	}

	// 直接在miniredis中设置值
	s.Set(testKey, string(data))

	// 测试获取存在的键
	var result string
	found, err := rs.Get(ctx, testKey, &result)
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
	if !found {
		t.Error("Get() should find the key")
	}
	if result != testValue {
		t.Errorf("Get() = %v, want %v", result, testValue)
	}

	// 测试获取不存在的键
	var result2 string
	found, err = rs.Get(ctx, "nonexistent_key", &result2)
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
	if found {
		t.Error("Get() should not find the key")
	}
}

func TestRedisStore_MGet(t *testing.T) {
	// 启动miniredis服务器
	s := miniredis.RunT(t)
	defer s.Close()

	// 创建redis客户端
	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	// 创建RedisStore
	rs := NewRedisStore(client)

	// 准备测试数据
	ctx := context.Background()
	testData := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	// 在miniredis中设置值
	for key, value := range testData {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("Failed to marshal test value: %v", err)
		}
		s.Set(key, string(data))
	}

	// 测试批量获取
	keys := []string{"key1", "key2", "nonexistent"}
	result := make(map[string]string)
	err := rs.MGet(ctx, keys, &result)
	if err != nil {
		t.Errorf("MGet() error = %v", err)
	}

	if len(result) != 2 {
		t.Errorf("MGet() should return 2 items, got %d", len(result))
	}
	if result["key1"] != "value1" {
		t.Errorf("MGet() key1 = %v, want %v", result["key1"], "value1")
	}
	if result["key2"] != "value2" {
		t.Errorf("MGet() key2 = %v, want %v", result["key2"], "value2")
	}
	if _, ok := result["nonexistent"]; ok {
		t.Error("MGet() should not return nonexistent key")
	}
}

func TestRedisStore_Exists(t *testing.T) {
	// 启动miniredis服务器
	s := miniredis.RunT(t)
	defer s.Close()

	// 创建redis客户端
	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	// 创建RedisStore
	rs := NewRedisStore(client)

	// 准备测试数据
	ctx := context.Background()
	testKey := "test_key"
	testValue := "test_value"

	// 序列化值
	data, err := json.Marshal(testValue)
	if err != nil {
		t.Fatalf("Failed to marshal test value: %v", err)
	}

	// 在miniredis中设置值
	s.Set(testKey, string(data))

	// 测试检查键存在性
	keys := []string{"test_key", "nonexistent"}
	result, err := rs.Exists(ctx, keys)
	if err != nil {
		t.Errorf("Exists() error = %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Exists() should return 2 items, got %d", len(result))
	}
	if !result["test_key"] {
		t.Error("Exists() should find test_key")
	}
	if result["nonexistent"] {
		t.Error("Exists() should not find nonexistent key")
	}
}

func TestRedisStore_MSet(t *testing.T) {
	// 启动miniredis服务器
	s := miniredis.RunT(t)
	defer s.Close()

	// 创建redis客户端
	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	// 创建RedisStore
	rs := NewRedisStore(client)

	// 准备测试数据
	ctx := context.Background()
	testData := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}

	// 测试批量设置
	err := rs.MSet(ctx, testData, 0)
	if err != nil {
		t.Errorf("MSet() error = %v", err)
	}

	// 验证设置的值
	for key, expected := range testData {
		data, err := s.Get(key)
		if err != nil {
			t.Errorf("Failed to get key %s from miniredis: %v", key, err)
			continue
		}

		var result string
		err = json.Unmarshal([]byte(data), &result)
		if err != nil {
			t.Errorf("Failed to unmarshal value for key %s: %v", key, err)
			continue
		}

		if result != expected {
			t.Errorf("MSet() key %s = %v, want %v", key, result, expected)
		}
	}

	// 测试带TTL的批量设置
	ttlData := map[string]interface{}{
		"ttl_key1": "ttl_value1",
		"ttl_key2": "ttl_value2",
	}
	ttl := 5 * time.Second

	err = rs.MSet(ctx, ttlData, ttl)
	if err != nil {
		t.Errorf("MSet() error = %v", err)
	}

	// 验证TTL设置
	for key := range ttlData {
		ttl := s.TTL(key)
		if ttl <= 0 {
			t.Errorf("MSet() should set TTL for key %s", key)
		}
	}
}

func TestRedisStore_Del(t *testing.T) {
	// 启动miniredis服务器
	s := miniredis.RunT(t)
	defer s.Close()

	// 创建redis客户端
	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	// 创建RedisStore
	rs := NewRedisStore(client)

	// 准备测试数据
	ctx := context.Background()
	testData := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	// 在miniredis中设置值
	for key, value := range testData {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("Failed to marshal test value: %v", err)
		}
		s.Set(key, string(data))
	}

	// 测试删除
	keys := []string{"key1", "key2", "nonexistent"}
	count, err := rs.Del(ctx, keys...)
	if err != nil {
		t.Errorf("Del() error = %v", err)
	}
	if count != 2 {
		t.Errorf("Del() should delete 2 keys, got %d", count)
	}

	// 验证删除结果
	if s.Exists("key1") {
		t.Error("Del() should delete key1")
	}
	if s.Exists("key2") {
		t.Error("Del() should delete key2")
	}
	if !s.Exists("key3") {
		t.Error("Del() should not delete key3")
	}
}

// 测试RedisStore实现Store接口
func TestRedisStore_Interface(t *testing.T) {
	// 启动miniredis服务器
	s := miniredis.RunT(t)
	defer s.Close()

	// 创建redis客户端
	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	// 创建RedisStore
	var _ store.Store = NewRedisStore(client)
}
