package cache

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// MockStore 实现Store接口用于测试
type MockStore struct {
	data map[string]interface{}
}

func NewMockStore() *MockStore {
	return &MockStore{
		data: make(map[string]interface{}),
	}
}

func (m *MockStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	if val, exists := m.data[key]; exists {
		// Simple copy for testing purposes
		dstValue := reflect.ValueOf(dst)
		if dstValue.Kind() == reflect.Ptr {
			dstValue.Elem().Set(reflect.ValueOf(val))
		}
		return true, nil
	}
	return false, nil
}

func (m *MockStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	mapValue := reflect.ValueOf(dstMap)
	if mapValue.Kind() != reflect.Ptr || mapValue.Elem().Kind() != reflect.Map {
		return assert.AnError
	}

	mapElem := mapValue.Elem()
	
	for _, key := range keys {
		if val, exists := m.data[key]; exists {
			mapElem.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(val))
		}
	}
	
	return nil
}

func (m *MockStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool)
	for _, key := range keys {
		_, exists := m.data[key]
		result[key] = exists
	}
	return result, nil
}

func (m *MockStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	for key, value := range items {
		m.data[key] = value
	}
	return nil
}

func (m *MockStore) Del(ctx context.Context, keys ...string) (int64, error) {
	count := int64(0)
	for _, key := range keys {
		if _, exists := m.data[key]; exists {
			delete(m.data, key)
			count++
		}
	}
	return count, nil
}

func TestMockStore(t *testing.T) {
	ctx := context.Background()
	store := NewMockStore()
	
	// Test MSet and Get
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	
	err := store.MSet(ctx, items, 0)
	assert.NoError(t, err)
	
	// Test Get
	var val string
	found, err := store.Get(ctx, "key1", &val)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "value1", val)
	
	// Test non-existent key
	found, err = store.Get(ctx, "nonexistent", &val)
	assert.NoError(t, err)
	assert.False(t, found)
	
	// Test MGet
	result := make(map[string]string)
	err = store.MGet(ctx, []string{"key1", "key2", "nonexistent"}, &result)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "value1", result["key1"])
	assert.Equal(t, "value2", result["key2"])
	
	// Test Exists
	exists, err := store.Exists(ctx, []string{"key1", "key2", "nonexistent"})
	assert.NoError(t, err)
	assert.True(t, exists["key1"])
	assert.True(t, exists["key2"])
	assert.False(t, exists["nonexistent"])
	
	// Test Del
	deleted, err := store.Del(ctx, "key1", "nonexistent")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), deleted)
	
	// Verify deletion
	found, err = store.Get(ctx, "key1", &val)
	assert.NoError(t, err)
	assert.False(t, found)
}

func TestCacher(t *testing.T) {
	ctx := context.Background()
	store := NewMockStore()
	cacher := NewCacher(store)
	
	// Define fallback functions
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		return "fallback_value_" + key, true, nil
	}
	
	batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		result := make(map[string]interface{})
		for _, key := range keys {
			result[key] = "batch_fallback_value_" + key
		}
		return result, nil
	}
	
	// Test Get with cache miss and fallback
	var val string
	found, err := cacher.Get(ctx, "test_key", &val, fallback, nil)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "fallback_value_test_key", val)
	
	// Test Get with cache hit
	found, err = cacher.Get(ctx, "test_key", &val, fallback, nil)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "fallback_value_test_key", val)
	
	// Test MGet with partial cache hit
	result := make(map[string]string)
	err = cacher.MGet(ctx, []string{"test_key", "nonexistent_key"}, &result, batchFallback, nil)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "fallback_value_test_key", result["test_key"])
	assert.Equal(t, "batch_fallback_value_nonexistent_key", result["nonexistent_key"])
	
	// Test MDelete
	deleted, err := cacher.MDelete(ctx, []string{"test_key"})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), deleted)
	
	// Verify deletion
	found, err = cacher.Get(ctx, "test_key", &val, fallback, nil)
	assert.NoError(t, err)
	assert.True(t, found) // Should fallback again
	assert.Equal(t, "fallback_value_test_key", val)
	
	// Test MRefresh
	result = make(map[string]string)
	err = cacher.MRefresh(ctx, []string{"test_key"}, &result, batchFallback, nil)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "fallback_value_test_key", result["test_key"])
}