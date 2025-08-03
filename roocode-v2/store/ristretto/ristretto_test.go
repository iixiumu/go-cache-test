package ristretto

import (
	"context"
	"testing"
	"time"

	"go-cache/store"

	"github.com/dgraph-io/ristretto/v2"
)

func TestRistrettoStore_Get(t *testing.T) {
	// 创建ristretto缓存
	cache, err := ristretto.NewCache(&ristretto.Config[string, interface{}]{
		NumCounters: 1000,
		MaxCost:     1000,
		BufferItems: 64,
	})
	if err != nil {
		t.Fatalf("Failed to create ristretto cache: %v", err)
	}
	defer cache.Close()

	// 创建RistrettoStore
	rs := NewRistrettoStore(cache)

	// 准备测试数据
	ctx := context.Background()
	testKey := "test_key"
	testValue := "test_value"

	// 在缓存中设置值
	cache.Set(testKey, testValue, 1)
	cache.Wait()

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

func TestRistrettoStore_MGet(t *testing.T) {
	// 创建ristretto缓存
	cache, err := ristretto.NewCache(&ristretto.Config[string, interface{}]{
		NumCounters: 1000,
		MaxCost:     1000,
		BufferItems: 64,
	})
	if err != nil {
		t.Fatalf("Failed to create ristretto cache: %v", err)
	}
	defer cache.Close()

	// 创建RistrettoStore
	rs := NewRistrettoStore(cache)

	// 准备测试数据
	ctx := context.Background()
	testData := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	// 在缓存中设置值
	for key, value := range testData {
		cache.Set(key, value, 1)
	}
	cache.Wait()

	// 测试批量获取
	keys := []string{"key1", "key2", "nonexistent"}
	result := make(map[string]string)
	err = rs.MGet(ctx, keys, &result)
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

func TestRistrettoStore_Exists(t *testing.T) {
	// 创建ristretto缓存
	cache, err := ristretto.NewCache(&ristretto.Config[string, interface{}]{
		NumCounters: 1000,
		MaxCost:     1000,
		BufferItems: 64,
	})
	if err != nil {
		t.Fatalf("Failed to create ristretto cache: %v", err)
	}
	defer cache.Close()

	// 创建RistrettoStore
	rs := NewRistrettoStore(cache)

	// 准备测试数据
	ctx := context.Background()
	testKey := "test_key"
	testValue := "test_value"

	// 在缓存中设置值
	cache.Set(testKey, testValue, 1)
	cache.Wait()

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

func TestRistrettoStore_MSet(t *testing.T) {
	// 创建ristretto缓存
	cache, err := ristretto.NewCache(&ristretto.Config[string, interface{}]{
		NumCounters: 1000,
		MaxCost:     1000,
		BufferItems: 64,
	})
	if err != nil {
		t.Fatalf("Failed to create ristretto cache: %v", err)
	}
	defer cache.Close()

	// 创建RistrettoStore
	rs := NewRistrettoStore(cache)

	// 准备测试数据
	ctx := context.Background()
	testData := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}

	// 测试批量设置
	err = rs.MSet(ctx, testData, 0)
	if err != nil {
		t.Errorf("MSet() error = %v", err)
	}

	// 等待值被缓冲
	cache.Wait()

	// 验证设置的值
	for key, expected := range testData {
		value, found := cache.Get(key)
		if !found {
			t.Errorf("MSet() should set key %s", key)
			continue
		}

		if value != expected {
			t.Errorf("MSet() key %s = %v, want %v", key, value, expected)
		}
	}

	// 测试带TTL的批量设置
	ttlData := map[string]interface{}{
		"ttl_key1": "ttl_value1",
		"ttl_key2": "ttl_value2",
	}
	ttl := 5 * time.Second

	// Ristretto不直接支持TTL，但我们可以在值中包含过期时间信息
	err = rs.MSet(ctx, ttlData, ttl)
	if err != nil {
		t.Errorf("MSet() error = %v", err)
	}

	// 等待值被缓冲
	cache.Wait()

	// 验证设置的值
	for key, expected := range ttlData {
		value, found := cache.Get(key)
		if !found {
			t.Errorf("MSet() should set key %s", key)
			continue
		}

		if value != expected {
			t.Errorf("MSet() key %s = %v, want %v", key, value, expected)
		}
	}
}

func TestRistrettoStore_Del(t *testing.T) {
	// 创建ristretto缓存
	cache, err := ristretto.NewCache(&ristretto.Config[string, interface{}]{
		NumCounters: 1000,
		MaxCost:     1000,
		BufferItems: 64,
	})
	if err != nil {
		t.Fatalf("Failed to create ristretto cache: %v", err)
	}
	defer cache.Close()

	// 创建RistrettoStore
	rs := NewRistrettoStore(cache)

	// 准备测试数据
	ctx := context.Background()
	testData := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	// 在缓存中设置值
	for key, value := range testData {
		cache.Set(key, value, 1)
	}
	cache.Wait()

	// 测试删除
	keys := []string{"key1", "key2", "nonexistent"}
	_, err = rs.Del(ctx, keys...)
	if err != nil {
		t.Errorf("Del() error = %v", err)
	}
	// 验证删除结果
	if _, found := cache.Get("key1"); found {
		t.Error("Del() should delete key1")
	}
	if _, found := cache.Get("key2"); found {
		t.Error("Del() should delete key2")
	}
	if _, found := cache.Get("key3"); !found {
		t.Error("Del() should not delete key3")
	}
}

// 测试RistrettoStore实现Store接口
func TestRistrettoStore_Interface(t *testing.T) {
	// 创建ristretto缓存
	cache, err := ristretto.NewCache(&ristretto.Config[string, interface{}]{
		NumCounters: 1000,
		MaxCost:     1000,
		BufferItems: 64,
	})
	if err != nil {
		t.Fatalf("Failed to create ristretto cache: %v", err)
	}
	defer cache.Close()

	// 创建RistrettoStore
	var _ store.Store = NewRistrettoStore(cache)
}
