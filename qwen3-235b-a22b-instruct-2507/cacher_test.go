package cache

import (
	"context"
	"reflect"
	"testing"
	"time"
)

// MockStore is a mock implementation of the Store interface for testing
type MockStore struct {
	data      map[string]interface{}
	existsMap map[string]bool
	ttl       map[string]time.Duration
}

// NewMockStore creates a new mock store
func NewMockStore() *MockStore {
	return &MockStore{
		data:      make(map[string]interface{}),
		existsMap: make(map[string]bool),
		ttl:       make(map[string]time.Duration),
	}
}

func (m *MockStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	if val, exists := m.data[key]; exists {
		// Copy the value to dst using reflection
		v := reflect.ValueOf(dst)
		if v.Kind() != reflect.Ptr || v.IsNil() {
			return false, nil
		}
		v.Elem().Set(reflect.ValueOf(val))
		return true, nil
	}
	return false, nil
}

func (m *MockStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	v := reflect.ValueOf(dstMap)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return nil
	}
	mapVal := v.Elem()
	if mapVal.Kind() != reflect.Map {
		return nil
	}
	
	for _, key := range keys {
		if val, exists := m.data[key]; exists {
			mapVal.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(val))
		}
	}
	return nil
}

func (m *MockStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool)
	for _, key := range keys {
		result[key] = m.existsMap[key]
	}
	return result, nil
}

func (m *MockStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	for k, v := range items {
		m.data[k] = v
		m.existsMap[k] = true
		if ttl > 0 {
			m.ttl[k] = ttl
		}
	}
	return nil
}

func (m *MockStore) Del(ctx context.Context, keys ...string) (int64, error) {
	var deleted int64
	for _, key := range keys {
		if _, exists := m.data[key]; exists {
			delete(m.data, key)
			delete(m.existsMap, key)
			delete(m.ttl, key)
			deleted++
		}
	}
	return deleted, nil
}

// TestGet tests the Get method of Cacher
func TestGet(t *testing.T) {
	store := NewMockStore()
	cacher := NewCacher(store)
	
	// Test cache hit
	store.data["key1"] = "value1"
	var result string
	found, err := cacher.Get(context.Background(), "key1", &result, nil, nil)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Errorf("Expected to find key1, but it was not found")
	}
	if result != "value1" {
		t.Errorf("Expected value 'value1', got '%s'", result)
	}
	
	// Test cache miss with fallback
	var fallbackCalled bool
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		fallbackCalled = true
		if key == "key2" {
			return "fallback_value", true, nil
		}
		return nil, false, nil
	}
	
	var result2 string
	found, err = cacher.Get(context.Background(), "key2", &result2, fallback, nil)
	if err != nil {
		t.Fatalf("Get with fallback failed: %v", err)
	}
	if !found {
		t.Errorf("Expected to find key2 via fallback, but it was not found")
	}
	if !fallbackCalled {
		t.Errorf("Fallback function was not called")
	}
	if result2 != "fallback_value" {
		t.Errorf("Expected fallback value 'fallback_value', got '%s'", result2)
	}
}

// TestMGet tests the MGet method of Cacher
func TestMGet(t *testing.T) {
	store := NewMockStore()
	cacher := NewCacher(store)
	
	// Set up some data in the store
	store.data["key1"] = "value1"
	store.data["key2"] = "value2"
	
	// Test partial cache hit
	var resultMap map[string]string
	err := cacher.MGet(context.Background(), []string{"key1", "key2", "key3"}, &resultMap, nil, nil)
	if err != nil {
		t.Fatalf("MGet failed: %v", err)
	}
	if len(resultMap) != 2 {
		t.Errorf("Expected 2 results, got %d", len(resultMap))
	}
	if resultMap["key1"] != "value1" {
		t.Errorf("Expected value 'value1' for key1, got '%s'", resultMap["key1"])
	}
	if resultMap["key2"] != "value2" {
		t.Errorf("Expected value 'value2' for key2, got '%s'", resultMap["key2"])
	}
	
	// Test with fallback for missing keys
	var fallbackCalled bool
	batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		fallbackCalled = true
		result := make(map[string]interface{})
		for _, key := range keys {
			if key == "key3" {
				result[key] = "fallback_value"
			}
		}
		return result, nil
	}
	
	var resultMap2 map[string]string
	err = cacher.MGet(context.Background(), []string{"key1", "key3"}, &resultMap2, batchFallback, nil)
	if err != nil {
		t.Fatalf("MGet with fallback failed: %v", err)
	}
	if !fallbackCalled {
		t.Errorf("Batch fallback function was not called")
	}
	if len(resultMap2) != 2 {
		t.Errorf("Expected 2 results, got %d", len(resultMap2))
	}
	if resultMap2["key3"] != "fallback_value" {
		t.Errorf("Expected fallback value 'fallback_value' for key3, got '%s'", resultMap2["key3"])
	}
}

// TestMDelete tests the MDelete method of Cacher
func TestMDelete(t *testing.T) {
	store := NewMockStore()
	cacher := NewCacher(store)
	
	// Set up data
	store.data["key1"] = "value1"
	store.data["key2"] = "value2"
	
	// Delete keys
	deleted, err := cacher.MDelete(context.Background(), []string{"key1", "key3"})
	if err != nil {
		t.Fatalf("MDelete failed: %v", err)
	}
	if deleted != 1 {
		t.Errorf("Expected 1 deletion, got %d", deleted)
	}
	
	// Verify key1 is deleted
	var result string
	found, _ := store.Get(context.Background(), "key1", &result)
	if found {
		t.Errorf("Expected key1 to be deleted, but it still exists")
	}
}

// TestMRefresh tests the MRefresh method of Cacher
func TestMRefresh(t *testing.T) {
	store := NewMockStore()
	cacher := NewCacher(store)
	
	// Set up initial data
	store.data["key1"] = "old_value"
	
	// Test refresh with fallback
	var fallbackCalled bool
	batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		fallbackCalled = true
		result := make(map[string]interface{})
		for _, key := range keys {
			result[key] = "new_value"
		}
		return result, nil
	}
	
	var resultMap map[string]string
	err := cacher.MRefresh(context.Background(), []string{"key1"}, &resultMap, batchFallback, nil)
	if err != nil {
		t.Fatalf("MRefresh failed: %v", err)
	}
	if !fallbackCalled {
		t.Errorf("Batch fallback function was not called")
	}
	if len(resultMap) != 1 {
		t.Errorf("Expected 1 result, got %d", len(resultMap))
	}
	if resultMap["key1"] != "new_value" {
		t.Errorf("Expected value 'new_value', got '%s'", resultMap["key1"])
	}
	
	// Verify the store has the new value
	var result string
	found, _ := store.Get(context.Background(), "key1", &result)
	if !found {
		t.Errorf("Expected key1 to exist in store, but it doesn't")
	}
	if result != "new_value" {
		t.Errorf("Expected value 'new_value' in store, got '%s'", result)
	}
}