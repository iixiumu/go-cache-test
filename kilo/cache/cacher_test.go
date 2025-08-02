package cache

import (
	"context"
	"testing"
	"time"
)

// mockStore 是一个模拟的Store实现，用于测试
type mockStore struct {
	data map[string][]byte
}

func newMockStore() *mockStore {
	return &mockStore{
		data: make(map[string][]byte),
	}
}

func (m *mockStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	// 简单实现，实际应该反序列化
	val, exists := m.data[key]
	if !exists {
		return false, nil
	}
	*dst.(*string) = string(val)
	return true, nil
}

func (m *mockStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	// 简单实现
	result := dstMap.(*map[string]string)
	for _, key := range keys {
		if val, exists := m.data[key]; exists {
			(*result)[key] = string(val)
		}
	}
	return nil
}

func (m *mockStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool)
	for _, key := range keys {
		_, exists := m.data[key]
		result[key] = exists
	}
	return result, nil
}

func (m *mockStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	for key, value := range items {
		m.data[key] = []byte(value.(string))
	}
	return nil
}

func (m *mockStore) Del(ctx context.Context, keys ...string) (int64, error) {
	var deleted int64
	for _, key := range keys {
		if _, exists := m.data[key]; exists {
			delete(m.data, key)
			deleted++
		}
	}
	return deleted, nil
}

func TestCacher_Get(t *testing.T) {
	store := newMockStore()
	cacher := NewCacher(store)

	// 测试缓存命中
	ctx := context.Background()
	key := "test_key"
	value := "test_value"
	store.MSet(ctx, map[string]interface{}{key: value}, 0)

	var result string
	found, err := cacher.Get(ctx, key, &result, nil, nil)
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

func TestCacher_GetWithFallback(t *testing.T) {
	store := newMockStore()
	cacher := NewCacher(store)

	// 测试缓存未命中，使用回退函数
	ctx := context.Background()
	key := "test_key"
	value := "fallback_value"

	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		return value, true, nil
	}

	var result string
	found, err := cacher.Get(ctx, key, &result, fallback, nil)
	if err != nil {
		t.Errorf("Get returned error: %v", err)
	}
	if !found {
		t.Error("Get should have found the key via fallback")
	}
	if result != value {
		t.Errorf("Get returned wrong value: got %v, want %v", result, value)
	}

	// 验证值已被缓存
	var result2 string
	found2, err2 := cacher.Get(ctx, key, &result2, nil, nil)
	if err2 != nil {
		t.Errorf("Get returned error: %v", err2)
	}
	if !found2 {
		t.Error("Get should have found the key in cache")
	}
	if result2 != value {
		t.Errorf("Get returned wrong value: got %v, want %v", result2, value)
	}
}

func TestCacher_MGet(t *testing.T) {
	store := newMockStore()
	cacher := NewCacher(store)

	// 测试批量获取
	ctx := context.Background()
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	store.MSet(ctx, items, 0)

	keys := []string{"key1", "key2", "key3"}
	result := make(map[string]string)
	err := cacher.MGet(ctx, keys, &result, nil, nil)
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

func TestCacher_MDelete(t *testing.T) {
	store := newMockStore()
	cacher := NewCacher(store)

	// 测试批量删除
	ctx := context.Background()
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	store.MSet(ctx, items, 0)

	keys := []string{"key1", "key2"}
	deleted, err := cacher.MDelete(ctx, keys)
	if err != nil {
		t.Errorf("MDelete returned error: %v", err)
	}

	if deleted != 2 {
		t.Errorf("MDelete deleted wrong number of items: got %v, want %v", deleted, 2)
	}

	// 验证键已被删除
	for _, key := range keys {
		found, err := store.Get(ctx, key, new(string))
		if err != nil {
			t.Errorf("Store Get returned error: %v", err)
		}
		if found {
			t.Errorf("Key %v should have been deleted", key)
		}
	}
}

func TestCacher_MRefresh(t *testing.T) {
	store := newMockStore()
	cacher := NewCacher(store)

	// 测试批量刷新
	ctx := context.Background()
	keys := []string{"key1", "key2"}

	fallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		result := make(map[string]interface{})
		for _, key := range keys {
			result[key] = "refreshed_" + key
		}
		return result, nil
	}

	result := make(map[string]string)
	err := cacher.MRefresh(ctx, keys, &result, fallback, nil)
	if err != nil {
		t.Errorf("MRefresh returned error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("MRefresh returned wrong number of items: got %v, want %v", len(result), 2)
	}

	// 验证值已被缓存
	for _, key := range keys {
		var value string
		found, err := store.Get(ctx, key, &value)
		if err != nil {
			t.Errorf("Store Get returned error: %v", err)
		}
		if !found {
			t.Errorf("Key %v should have been cached", key)
		}
		if value != "refreshed_"+key {
			t.Errorf("Key %v has wrong value: got %v, want %v", key, value, "refreshed_"+key)
		}
	}
}
