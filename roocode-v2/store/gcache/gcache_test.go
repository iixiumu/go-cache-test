package gcache

import (
	"context"
	"testing"
	"time"

	"go-cache/store"

	"github.com/bluele/gcache"
)

func TestGCacheStore_Get(t *testing.T) {
	// 创建gcache缓存
	gc := gcache.New(100).
		LRU().
		Build()

	// 创建GCacheStore
	gs := NewGCacheStore(gc)

	// 准备测试数据
	ctx := context.Background()
	testKey := "test_key"
	testValue := "test_value"

	// 在缓存中设置值
	gc.Set(testKey, testValue)

	// 测试获取存在的键
	var result string
	found, err := gs.Get(ctx, testKey, &result)
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
	found, err = gs.Get(ctx, "nonexistent_key", &result2)
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
	if found {
		t.Error("Get() should not find the key")
	}
}

func TestGCacheStore_MGet(t *testing.T) {
	// 创建gcache缓存
	gc := gcache.New(100).
		LRU().
		Build()

	// 创建GCacheStore
	gs := NewGCacheStore(gc)

	// 准备测试数据
	ctx := context.Background()
	testData := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	// 在缓存中设置值
	for key, value := range testData {
		gc.Set(key, value)
	}

	// 测试批量获取
	keys := []string{"key1", "key2", "nonexistent"}
	result := make(map[string]string)
	err := gs.MGet(ctx, keys, &result)
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

func TestGCacheStore_Exists(t *testing.T) {
	// 创建gcache缓存
	gc := gcache.New(100).
		LRU().
		Build()

	// 创建GCacheStore
	gs := NewGCacheStore(gc)

	// 准备测试数据
	ctx := context.Background()
	testKey := "test_key"
	testValue := "test_value"

	// 在缓存中设置值
	gc.Set(testKey, testValue)

	// 测试检查键存在性
	keys := []string{"test_key", "nonexistent"}
	result, err := gs.Exists(ctx, keys)
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

func TestGCacheStore_MSet(t *testing.T) {
	// 创建gcache缓存
	gc := gcache.New(100).
		LRU().
		Build()

	// 创建GCacheStore
	gs := NewGCacheStore(gc)

	// 准备测试数据
	ctx := context.Background()
	testData := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}

	// 测试批量设置
	err := gs.MSet(ctx, testData, 0)
	if err != nil {
		t.Errorf("MSet() error = %v", err)
	}

	// 验证设置的值
	for key, expected := range testData {
		value, err := gc.Get(key)
		if err != nil {
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

	err = gs.MSet(ctx, ttlData, ttl)
	if err != nil {
		t.Errorf("MSet() error = %v", err)
	}

	// 验证设置的值
	for key, expected := range ttlData {
		value, err := gc.Get(key)
		if err != nil {
			t.Errorf("MSet() should set key %s", key)
			continue
		}

		if value != expected {
			t.Errorf("MSet() key %s = %v, want %v", key, value, expected)
		}
	}

	// 等待TTL过期
	time.Sleep(ttl + time.Second)

	// 验证值已过期
	for key := range ttlData {
		_, err := gc.Get(key)
		if err == nil {
			t.Errorf("MSet() key %s should expire", key)
		}
	}
}

func TestGCacheStore_Del(t *testing.T) {
	// 创建gcache缓存
	gc := gcache.New(100).
		LRU().
		Build()

	// 创建GCacheStore
	gs := NewGCacheStore(gc)

	// 准备测试数据
	ctx := context.Background()
	testData := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	// 在缓存中设置值
	for key, value := range testData {
		gc.Set(key, value)
	}

	// 测试删除
	keys := []string{"key1", "key2", "nonexistent"}
	count, err := gs.Del(ctx, keys...)
	if err != nil {
		t.Errorf("Del() error = %v", err)
	}
	if count != 2 {
		t.Errorf("Del() should delete 2 keys, got %d", count)
	}

	// 验证删除结果
	if _, err := gc.Get("key1"); err == nil {
		t.Error("Del() should delete key1")
	}
	if _, err := gc.Get("key2"); err == nil {
		t.Error("Del() should delete key2")
	}
	if _, err := gc.Get("key3"); err != nil {
		t.Error("Del() should not delete key3")
	}
}

// 测试GCacheStore实现Store接口
func TestGCacheStore_Interface(t *testing.T) {
	// 创建gcache缓存
	gc := gcache.New(100).
		LRU().
		Build()

	// 创建GCacheStore
	var _ store.Store = NewGCacheStore(gc)
}
