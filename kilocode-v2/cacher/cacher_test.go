package cacher

import (
	"context"
	"reflect"
	"testing"
	"time"

	"go-cache/cacher/store"
)

// mockStore 是一个模拟的存储实现，用于测试
type mockStore struct {
	data map[string]interface{}
}

func newMockStore() store.Store {
	return &mockStore{
		data: make(map[string]interface{}),
	}
}

func (m *mockStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	value, exists := m.data[key]
	if !exists {
		return false, nil
	}

	// 简单的类型赋值
	switch dst := dst.(type) {
	case *string:
		if str, ok := value.(string); ok {
			*dst = str
			return true, nil
		}
	case *int:
		if i, ok := value.(int); ok {
			*dst = i
			return true, nil
		}
	}

	return false, nil
}

func (m *mockStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	// 获取目标map的反射值
	dstValue := reflect.ValueOf(dstMap)
	if dstValue.Kind() != reflect.Ptr || dstValue.Elem().Kind() != reflect.Map {
		return &reflect.ValueError{Method: "MGet", Kind: dstValue.Kind()}
	}

	mapValue := dstValue.Elem()
	mapValue.Set(reflect.MakeMap(mapValue.Type()))

	// 填充结果
	for _, key := range keys {
		if value, exists := m.data[key]; exists {
			// 创建新元素
			elemType := mapValue.Type().Elem()
			elemValue := reflect.New(elemType).Elem()

			// 设置值
			srcValue := reflect.ValueOf(value)
			if srcValue.Type().AssignableTo(elemType) {
				elemValue.Set(srcValue)
			} else if srcValue.Type().ConvertibleTo(elemType) {
				elemValue.Set(srcValue.Convert(elemType))
			}

			// 设置map值
			mapValue.SetMapIndex(reflect.ValueOf(key), elemValue)
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

func TestCacherGet(t *testing.T) {
	// 创建模拟存储
	store := newMockStore()

	// 创建Cacher实例
	cacher := NewCacher(store)

	// 测试数据
	ctx := context.Background()
	key := "test_key"
	expectedValue := "test_value"

	// 测试缓存未命中时使用回退函数
	var result string
	fallbackCalled := false
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		fallbackCalled = true
		return expectedValue, true, nil
	}

	found, err := cacher.Get(ctx, key, &result, fallback, nil)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if !found {
		t.Error("Value should be found")
	}

	if !fallbackCalled {
		t.Error("Fallback function should be called")
	}

	if result != expectedValue {
		t.Errorf("Expected %v, got %v", expectedValue, result)
	}

	// 测试缓存命中时不再调用回退函数
	fallbackCalled = false
	found, err = cacher.Get(ctx, key, &result, fallback, nil)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if !found {
		t.Error("Value should be found")
	}

	if fallbackCalled {
		t.Error("Fallback function should not be called when value is in cache")
	}

	if result != expectedValue {
		t.Errorf("Expected %v, got %v", expectedValue, result)
	}
}

func TestCacherMGet(t *testing.T) {
	// 创建模拟存储
	store := newMockStore()

	// 创建Cacher实例
	cacher := NewCacher(store)

	// 测试数据
	ctx := context.Background()
	keys := []string{"key1", "key2", "key3"}

	// 先在缓存中设置一些值
	store.MSet(ctx, map[string]interface{}{
		"key1": "value1",
	}, 0)

	// 批量回退函数
	batchFallbackCalled := false
	batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		batchFallbackCalled = true
		result := make(map[string]interface{})
		for _, key := range keys {
			if key != "key1" { // key1已经在缓存中
				result[key] = "fallback_" + key
			}
		}
		return result, nil
	}

	// 执行批量获取
	result := make(map[string]string)
	err := cacher.MGet(ctx, keys, &result, batchFallback, nil)
	if err != nil {
		t.Fatalf("MGet failed: %v", err)
	}

	if !batchFallbackCalled {
		t.Error("Batch fallback function should be called")
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 items, got %d", len(result))
	}

	if result["key1"] != "value1" {
		t.Errorf("Expected value1, got %v", result["key1"])
	}

	if result["key2"] != "fallback_key2" {
		t.Errorf("Expected fallback_key2, got %v", result["key2"])
	}

	if result["key3"] != "fallback_key3" {
		t.Errorf("Expected fallback_key3, got %v", result["key3"])
	}
}

func TestCacherMDelete(t *testing.T) {
	// 创建模拟存储
	store := newMockStore()

	// 创建Cacher实例
	cacher := NewCacher(store)

	// 测试数据
	ctx := context.Background()

	// 先在缓存中设置一些值
	store.MSet(ctx, map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}, 0)

	// 删除部分键
	deleted, err := cacher.MDelete(ctx, []string{"key1", "key2"})
	if err != nil {
		t.Fatalf("MDelete failed: %v", err)
	}

	if deleted != 2 {
		t.Errorf("Expected 2 deleted keys, got %d", deleted)
	}

	// 验证键是否被删除
	var result string
	found, err := cacher.Get(ctx, "key1", &result, nil, nil)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if found {
		t.Error("key1 should not be found after deletion")
	}
}

func TestCacherMRefresh(t *testing.T) {
	// 创建模拟存储
	store := newMockStore()

	// 创建Cacher实例
	cacher := NewCacher(store)

	// 测试数据
	ctx := context.Background()
	keys := []string{"key1", "key2"}

	// 先在缓存中设置一些值
	store.MSet(ctx, map[string]interface{}{
		"key1": "old_value1",
		"key2": "old_value2",
	}, 0)

	// 强制刷新函数
	refreshFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		result := make(map[string]interface{})
		for _, key := range keys {
			result[key] = "new_" + key
		}
		return result, nil
	}

	// 执行强制刷新
	result := make(map[string]string)
	err := cacher.MRefresh(ctx, keys, &result, refreshFallback, nil)
	if err != nil {
		t.Fatalf("MRefresh failed: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 items, got %d", len(result))
	}

	if result["key1"] != "new_key1" {
		t.Errorf("Expected new_key1, got %v", result["key1"])
	}

	if result["key2"] != "new_key2" {
		t.Errorf("Expected new_key2, got %v", result["key2"])
	}

	// 验证缓存中的值是否已更新
	var cachedValue string
	found, err := cacher.Get(ctx, "key1", &cachedValue, nil, nil)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if !found {
		t.Error("key1 should be found")
	}

	if cachedValue != "new_key1" {
		t.Errorf("Expected new_key1, got %v", cachedValue)
	}
}
