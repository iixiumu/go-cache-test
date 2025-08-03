package cacher

import (
	"context"
	"errors"
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
	if val, ok := m.data[key]; ok {
		// 使用反射设置dst的值
		dstValue := reflect.ValueOf(dst)
		if dstValue.Kind() != reflect.Ptr {
			return false, errors.New("dst must be a pointer")
		}
		dstValue = dstValue.Elem()
		valValue := reflect.ValueOf(val)
		if dstValue.Type() != valValue.Type() {
			return false, errors.New("type mismatch")
		}
		dstValue.Set(valValue)
		return true, nil
	}
	return false, nil
}

func (m *mockStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr {
		return errors.New("dstMap must be a pointer")
	}
	dstMapValue = dstMapValue.Elem()
	if dstMapValue.Kind() != reflect.Map {
		return errors.New("dstMap must be a pointer to a map")
	}

	mapType := dstMapValue.Type()
	keyType := mapType.Key()
	valueType := mapType.Elem()

	newMap := reflect.MakeMap(mapType)
	for _, key := range keys {
		if val, ok := m.data[key]; ok {
			keyValue := reflect.ValueOf(key).Convert(keyType)
			valValue := reflect.ValueOf(val).Convert(valueType)
			newMap.SetMapIndex(keyValue, valValue)
		}
	}
	dstMapValue.Set(newMap)
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
		if _, ok := m.data[key]; ok {
			delete(m.data, key)
			count++
		}
	}
	return count, nil
}

// 测试用的回退函数
func testFallbackFunc(ctx context.Context, key string) (interface{}, bool, error) {
	if key == "error_key" {
		return nil, false, errors.New("fallback error")
	}
	return "fallback_value_for_" + key, true, nil
}

func testBatchFallbackFunc(ctx context.Context, keys []string) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	for _, key := range keys {
		if key == "error_key" {
			return nil, errors.New("batch fallback error")
		}
		result[key] = "batch_fallback_value_for_" + key
	}
	return result, nil
}

func TestCacher_Get(t *testing.T) {
	store := newMockStore()
	cacher := NewCacher(store)

	// 测试缓存命中
	ctx := context.Background()
	store.data["key1"] = "cached_value"
	var result string
	found, err := cacher.Get(ctx, "key1", &result, testFallbackFunc, nil)
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
	if !found {
		t.Error("Get() should find the key")
	}
	if result != "cached_value" {
		t.Errorf("Get() = %v, want %v", result, "cached_value")
	}

	// 测试缓存未命中，使用回退函数
	var result2 string
	found, err = cacher.Get(ctx, "key2", &result2, testFallbackFunc, nil)
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
	if !found {
		t.Error("Get() should find the key through fallback")
	}
	if result2 != "fallback_value_for_key2" {
		t.Errorf("Get() = %v, want %v", result2, "fallback_value_for_key2")
	}

	// 测试回退函数返回错误
	var result3 string
	found, err = cacher.Get(ctx, "error_key", &result3, testFallbackFunc, nil)
	if err == nil {
		t.Error("Get() should return error")
	}
	if found {
		t.Error("Get() should not find the key")
	}
}

func TestCacher_MGet(t *testing.T) {
	store := newMockStore()
	cacher := NewCacher(store)

	// 准备测试数据
	ctx := context.Background()
	store.data["key1"] = "value1"
	store.data["key2"] = "value2"

	// 测试部分命中
	result := make(map[string]string)
	keys := []string{"key1", "key2", "key3"}
	err := cacher.MGet(ctx, keys, &result, testBatchFallbackFunc, nil)
	if err != nil {
		t.Errorf("MGet() error = %v", err)
	}
	if len(result) != 3 {
		t.Errorf("MGet() should return 3 items, got %d", len(result))
	}
	if result["key1"] != "value1" {
		t.Errorf("MGet() key1 = %v, want %v", result["key1"], "value1")
	}
	if result["key2"] != "value2" {
		t.Errorf("MGet() key2 = %v, want %v", result["key2"], "value2")
	}
	if result["key3"] != "batch_fallback_value_for_key3" {
		t.Errorf("MGet() key3 = %v, want %v", result["key3"], "batch_fallback_value_for_key3")
	}

	// 测试回退函数返回错误
	result2 := make(map[string]string)
	keys2 := []string{"error_key"}
	err = cacher.MGet(ctx, keys2, &result2, testBatchFallbackFunc, nil)
	if err == nil {
		t.Error("MGet() should return error")
	}
}

func TestCacher_MDelete(t *testing.T) {
	store := newMockStore()
	cacher := NewCacher(store)

	// 准备测试数据
	ctx := context.Background()
	store.data["key1"] = "value1"
	store.data["key2"] = "value2"
	store.data["key3"] = "value3"

	// 测试删除
	keys := []string{"key1", "key2", "nonexistent"}
	count, err := cacher.MDelete(ctx, keys)
	if err != nil {
		t.Errorf("MDelete() error = %v", err)
	}
	if count != 2 {
		t.Errorf("MDelete() should delete 2 keys, got %d", count)
	}

	// 验证删除结果
	if _, ok := store.data["key1"]; ok {
		t.Error("MDelete() should delete key1")
	}
	if _, ok := store.data["key2"]; ok {
		t.Error("MDelete() should delete key2")
	}
	if _, ok := store.data["key3"]; !ok {
		t.Error("MDelete() should not delete key3")
	}
}

func TestCacher_MRefresh(t *testing.T) {
	store := newMockStore()
	cacher := NewCacher(store)

	// 准备测试数据
	ctx := context.Background()
	store.data["key1"] = "old_value1"
	store.data["key2"] = "old_value2"

	// 测试刷新
	result := make(map[string]string)
	keys := []string{"key1", "key2", "key3"}
	err := cacher.MRefresh(ctx, keys, &result, testBatchFallbackFunc, nil)
	if err != nil {
		t.Errorf("MRefresh() error = %v", err)
	}

	// 验证刷新结果
	if len(result) != 3 {
		t.Errorf("MRefresh() should return 3 items, got %d", len(result))
	}
	if result["key1"] != "batch_fallback_value_for_key1" {
		t.Errorf("MRefresh() key1 = %v, want %v", result["key1"], "batch_fallback_value_for_key1")
	}
	if result["key2"] != "batch_fallback_value_for_key2" {
		t.Errorf("MRefresh() key2 = %v, want %v", result["key2"], "batch_fallback_value_for_key2")
	}
	if result["key3"] != "batch_fallback_value_for_key3" {
		t.Errorf("MRefresh() key3 = %v, want %v", result["key3"], "batch_fallback_value_for_key3")
	}
}
