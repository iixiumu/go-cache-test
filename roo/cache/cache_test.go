package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/xiumu/go-cache/store"
)

func TestCacheGet(t *testing.T) {
	// 创建一个Ristretto缓存实例用于测试
	rcache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1000,
		MaxCost:     10000,
		BufferItems: 64,
	})
	if err != nil {
		t.Fatalf("Failed to create Ristretto cache: %v", err)
	}

	// 创建存储实例
	s := store.NewRistrettoStore(rcache)

	// 创建缓存实例
	c := New(s)
	ctx := context.Background()

	// 测试带回退函数的Get
	var value string
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		if key == "test_key" {
			return "fallback_value", true, nil
		}
		return nil, false, nil
	}

	found, err := c.Get(ctx, "test_key", &value, fallback, nil)
	if err != nil {
		t.Fatalf("Failed to Get: %v", err)
	}
	if !found {
		t.Fatalf("Key not found")
	}
	if value != "fallback_value" {
		t.Fatalf("Expected 'fallback_value', got '%s'", value)
	}

	// 等待一点时间让缓存写入完成
	time.Sleep(time.Millisecond * 10)

	// 验证值已缓存
	var cachedValue string
	found, err = c.Get(ctx, "test_key", &cachedValue, nil, nil)
	if err != nil {
		t.Fatalf("Failed to Get: %v", err)
	}
	if !found {
		t.Fatalf("Key not found in cache")
	}
	if cachedValue != "fallback_value" {
		t.Fatalf("Expected 'fallback_value', got '%s'", cachedValue)
	}
}

func TestCacheMGet(t *testing.T) {
	// 创建一个Ristretto缓存实例用于测试
	rcache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1000,
		MaxCost:     10000,
		BufferItems: 64,
	})
	if err != nil {
		t.Fatalf("Failed to create Ristretto cache: %v", err)
	}

	// 创建存储实例
	s := store.NewRistrettoStore(rcache)

	// 创建缓存实例
	c := New(s)
	ctx := context.Background()

	// 先预设一些值到缓存中
	items := map[string]interface{}{
		"key1": "value1",
	}
	err = s.MSet(ctx, items, time.Second*10)
	if err != nil {
		t.Fatalf("Failed to MSet: %v", err)
	}

	// 等待一点时间让缓存写入完成
	time.Sleep(time.Millisecond * 10)

	// 测试带回退函数的MGet
	result := make(map[string]string)
	keys := []string{"key1", "key2", "key3"}

	batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		result := make(map[string]interface{})
		for _, key := range keys {
			if key == "key2" {
				result[key] = "value2"
			} else if key == "key3" {
				result[key] = "value3"
			}
		}
		return result, nil
	}

	err = c.MGet(ctx, keys, &result, batchFallback, nil)
	if err != nil {
		t.Fatalf("Failed to MGet: %v", err)
	}

	// 检查结果
	if len(result) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(result))
	}
	if result["key1"] != "value1" {
		t.Fatalf("Expected 'value1', got '%s'", result["key1"])
	}
	if result["key2"] != "value2" {
		t.Fatalf("Expected 'value2', got '%s'", result["key2"])
	}
	if result["key3"] != "value3" {
		t.Fatalf("Expected 'value3', got '%s'", result["key3"])
	}
}

func TestCacheMDelete(t *testing.T) {
	// 创建一个Ristretto缓存实例用于测试
	rcache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1000,
		MaxCost:     10000,
		BufferItems: 64,
	})
	if err != nil {
		t.Fatalf("Failed to create Ristretto cache: %v", err)
	}

	// 创建存储实例
	s := store.NewRistrettoStore(rcache)

	// 创建缓存实例
	c := New(s)
	ctx := context.Background()

	// 预设一些值到缓存中
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	err = s.MSet(ctx, items, time.Second*10)
	if err != nil {
		t.Fatalf("Failed to MSet: %v", err)
	}

	// 等待一点时间让缓存写入完成
	time.Sleep(time.Millisecond * 10)

	// 测试MDelete
	count, err := c.MDelete(ctx, []string{"key1", "key2"})
	if err != nil {
		t.Fatalf("Failed to MDelete: %v", err)
	}
	if count != 2 {
		t.Fatalf("Expected to delete 2 keys, got %d", count)
	}

	// 验证键已被删除
	var value string
	found, err := s.Get(ctx, "key1", &value)
	if err != nil {
		t.Fatalf("Failed to Get: %v", err)
	}
	if found {
		t.Fatalf("Key should have been deleted")
	}
}

func TestCacheMRefresh(t *testing.T) {
	// 创建一个Ristretto缓存实例用于测试
	rcache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1000,
		MaxCost:     10000,
		BufferItems: 64,
	})
	if err != nil {
		t.Fatalf("Failed to create Ristretto cache: %v", err)
	}

	// 创建存储实例
	s := store.NewRistrettoStore(rcache)

	// 创建缓存实例
	c := New(s)
	ctx := context.Background()

	// 预设一些值到缓存中
	items := map[string]interface{}{
		"key1": "old_value1",
		"key2": "old_value2",
	}
	err = s.MSet(ctx, items, time.Second*10)
	if err != nil {
		t.Fatalf("Failed to MSet: %v", err)
	}

	// 等待一点时间让缓存写入完成
	time.Sleep(time.Millisecond * 10)

	// 测试MRefresh
	result := make(map[string]string)
	keys := []string{"key1", "key2", "key3"}

	batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		result := make(map[string]interface{})
		for _, key := range keys {
			if key == "key1" {
				result[key] = "new_value1"
			} else if key == "key2" {
				result[key] = "new_value2"
			} else if key == "key3" {
				result[key] = "new_value3"
			}
		}
		return result, nil
	}

	err = c.MRefresh(ctx, keys, &result, batchFallback, nil)
	if err != nil {
		t.Fatalf("Failed to MRefresh: %v", err)
	}

	// 检查结果
	if len(result) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(result))
	}
	if result["key1"] != "new_value1" {
		t.Fatalf("Expected 'new_value1', got '%s'", result["key1"])
	}
	if result["key2"] != "new_value2" {
		t.Fatalf("Expected 'new_value2', got '%s'", result["key2"])
	}
	if result["key3"] != "new_value3" {
		t.Fatalf("Expected 'new_value3', got '%s'", result["key3"])
	}

	// 验证缓存中的值已更新
	var cachedValue string
	found, err := s.Get(ctx, "key1", &cachedValue)
	if err != nil {
		t.Fatalf("Failed to Get: %v", err)
	}
	if !found {
		t.Fatalf("Key not found in cache")
	}
	if cachedValue != "new_value1" {
		t.Fatalf("Expected 'new_value1', got '%s'", cachedValue)
	}
}

func TestCacheGetWithError(t *testing.T) {
	// 创建一个Ristretto缓存实例用于测试
	rcache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1000,
		MaxCost:     10000,
		BufferItems: 64,
	})
	if err != nil {
		t.Fatalf("Failed to create Ristretto cache: %v", err)
	}

	// 创建存储实例
	s := store.NewRistrettoStore(rcache)

	// 创建缓存实例
	c := New(s)
	ctx := context.Background()

	// 测试带错误的回退函数
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		return nil, false, errors.New("fallback error")
	}

	var value string
	_, err = c.Get(ctx, "error_key", &value, fallback, nil)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
}
