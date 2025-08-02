package cacher

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"
)

// mockStore 是一个模拟的Store实现，用于测试
type mockStore struct {
	data map[string]interface{}
}

func newMockStore() *mockStore {
	return &mockStore{
		data: make(map[string]interface{}),
	}
}

func (m *mockStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	val, exists := m.data[key]
	if !exists {
		return false, nil
	}

	// 使用反射来处理不同的类型
	// 这里简化处理，实际实现可能需要更复杂的反射逻辑
	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return false, fmt.Errorf("dst must be a non-nil pointer")
	}

	rv = rv.Elem()
	if !rv.CanSet() {
		return false, fmt.Errorf("cannot set dst value")
	}

	valRV := reflect.ValueOf(val)
	if !valRV.Type().AssignableTo(rv.Type()) {
		return false, fmt.Errorf("cannot assign %T to %T", val, dst)
	}

	rv.Set(valRV)
	return true, nil
}

func (m *mockStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	// 使用反射来处理不同的map类型
	// 这里简化处理，实际实现可能需要更复杂的反射逻辑
	rv := reflect.ValueOf(dstMap)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("dstMap must be a non-nil pointer")
	}

	rv = rv.Elem()
	if rv.Kind() != reflect.Map {
		return fmt.Errorf("dstMap must be a pointer to a map")
	}

	if rv.IsNil() {
		rv.Set(reflect.MakeMap(rv.Type()))
	}

	// 获取map的键和值类型
	mapType := rv.Type()
	keyType := mapType.Key()
	valueType := mapType.Elem()

	for _, key := range keys {
		if val, exists := m.data[key]; exists {
			// 创建键和值的反射值
			keyRV := reflect.ValueOf(key)
			if !keyRV.Type().AssignableTo(keyType) {
				return fmt.Errorf("cannot assign key %T to %v", key, keyType)
			}

			valRV := reflect.ValueOf(val)
			if !valRV.Type().AssignableTo(valueType) {
				return fmt.Errorf("cannot assign value %T to %v", val, valueType)
			}

			// 设置map中的值
			rv.SetMapIndex(keyRV, valRV)
		}
	}
	return nil
}

func (m *mockStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool)
	for _, key := range keys {
		_, result[key] = m.data[key]
	}
	return result, nil
}

func (m *mockStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	for key, value := range items {
		m.data[key] = value
	}
	return nil
}

func (m *mockStore) Del(ctx context.Context, keys ...string) (int64, error) {
	count := int64(0)
	for _, key := range keys {
		if _, exists := m.data[key]; exists {
			delete(m.data, key)
			count++
		}
	}
	return count, nil
}

// TestCacher_Get 测试Cacher的Get方法
func TestCacher_Get(t *testing.T) {
	mockStore := newMockStore()
	cacher := NewCacher(mockStore)

	ctx := context.Background()

	// 测试缓存命中
	key := "test_key"
	value := "test_value"
	mockStore.MSet(ctx, map[string]interface{}{key: value}, 0)

	var result string
	found, err := cacher.Get(ctx, key, &result, nil, nil)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}

	if !found {
		t.Error("Get should have found the key")
	}

	if result != value {
		t.Errorf("Get returned wrong value: got %v, want %v", result, value)
	}

	// 测试缓存未命中但有回退函数
	key2 := "test_key2"
	fallbackValue := "fallback_value"
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		return fallbackValue, true, nil
	}

	var result2 string
	found, err = cacher.Get(ctx, key2, &result2, fallback, nil)
	if err != nil {
		t.Fatalf("Get with fallback returned error: %v", err)
	}

	if !found {
		t.Error("Get with fallback should have found the key")
	}

	if result2 != fallbackValue {
		t.Errorf("Get with fallback returned wrong value: got %v, want %v", result2, fallbackValue)
	}

	// 验证回退值已存入缓存
	var result3 string
	found, err = mockStore.Get(ctx, key2, &result3)
	if err != nil {
		t.Fatalf("Store.Get returned error: %v", err)
	}

	if !found {
		t.Error("Fallback value should have been stored in cache")
	}

	if result3 != fallbackValue {
		t.Errorf("Store.Get returned wrong value: got %v, want %v", result3, fallbackValue)
	}
}

// TestCacher_MGet 测试Cacher的MGet方法
func TestCacher_MGet(t *testing.T) {
	store := newMockStore()
	cacher := NewCacher(store)

	ctx := context.Background()

	// 预设一些缓存数据
	store.MSet(ctx, map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}, 0)

	// 测试批量获取
	keys := []string{"key1", "key2", "key3"}
	result := make(map[string]string)

	fallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		return map[string]interface{}{
			"key3": "value3",
		}, nil
	}

	err := cacher.MGet(ctx, keys, &result, fallback, nil)
	if err != nil {
		t.Fatalf("MGet returned error: %v", err)
	}

	expected := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	for key, expectedValue := range expected {
		if value, exists := result[key]; !exists || value != expectedValue {
			t.Errorf("MGet result mismatch for key %s: got %v, want %v", key, value, expectedValue)
		}
	}
}

// TestCacher_MDelete 测试Cacher的MDelete方法
func TestCacher_MDelete(t *testing.T) {
	store := newMockStore()
	cacher := NewCacher(store)

	ctx := context.Background()

	// 预设一些缓存数据
	store.MSet(ctx, map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}, 0)

	// 删除部分键
	keys := []string{"key1", "key3"}
	deleted, err := cacher.MDelete(ctx, keys)
	if err != nil {
		t.Fatalf("MDelete returned error: %v", err)
	}

	if deleted != 2 {
		t.Errorf("MDelete returned wrong count: got %d, want %d", deleted, 2)
	}

	// 验证键已被删除
	for _, key := range keys {
		var result interface{}
		found, err := store.Get(ctx, key, &result)
		if err != nil {
			t.Fatalf("Store.Get returned error: %v", err)
		}

		if found {
			t.Errorf("Key %s should have been deleted", key)
		}
	}

	// 验证未删除的键仍然存在
	var result string
	found, err := store.Get(ctx, "key2", &result)
	if err != nil {
		t.Fatalf("Store.Get returned error: %v", err)
	}

	if !found {
		t.Error("Key key2 should still exist")
	}

	if result != "value2" {
		t.Errorf("Key key2 has wrong value: got %v, want %v", result, "value2")
	}
}

// TestCacher_MRefresh 测试Cacher的MRefresh方法
func TestCacher_MRefresh(t *testing.T) {
	store := newMockStore()
	cacher := NewCacher(store)

	ctx := context.Background()

	// 预设一些缓存数据
	store.MSet(ctx, map[string]interface{}{
		"key1": "old_value1",
		"key2": "old_value2",
	}, 0)

	// 刷新键
	keys := []string{"key1", "key3"}

	fallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		return map[string]interface{}{
			"key1": "new_value1",
			"key3": "new_value3",
		}, nil
	}

	result := make(map[string]string)
	err := cacher.MRefresh(ctx, keys, &result, fallback, nil)
	if err != nil {
		t.Fatalf("MRefresh returned error: %v", err)
	}

	// 验证缓存已被更新
	expected := map[string]string{
		"key1": "new_value1",
		"key3": "new_value3",
	}

	for key, expectedValue := range expected {
		var value string
		found, err := store.Get(ctx, key, &value)
		if err != nil {
			t.Fatalf("Store.Get returned error: %v", err)
		}

		if !found {
			t.Errorf("Key %s should exist after refresh", key)
		}

		if value != expectedValue {
			t.Errorf("Key %s has wrong value: got %v, want %v", key, value, expectedValue)
		}
	}

	// 验证key2仍然存在（因为它不在刷新列表中）
	var result2 interface{}
	found, err := store.Get(ctx, "key2", &result2)
	if err != nil {
		t.Fatalf("Store.Get returned error: %v", err)
	}

	if !found {
		t.Error("Key key2 should still exist after refresh")
	}
}
