package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockStore is a mock implementation of the Store interface for testing
type MockStore struct {
	mock.Mock
}

func (m *MockStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	args := m.Called(ctx, key, dst)
	return args.Bool(0), args.Error(1)
}

func (m *MockStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	args := m.Called(ctx, keys, dstMap)
	return args.Error(0)
}

func (m *MockStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	args := m.Called(ctx, keys)
	return args.Get(0).(map[string]bool), args.Error(1)
}

func (m *MockStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	args := m.Called(ctx, items, ttl)
	return args.Error(0)
}

func (m *MockStore) Del(ctx context.Context, keys ...string) (int64, error) {
	args := m.Called(ctx, keys)
	return args.Get(0).(int64), args.Error(1)
}

func TestNewCacher(t *testing.T) {
	mockStore := &MockStore{}
	cacher := NewCacher(mockStore)
	
	assert.NotNil(t, cacher)
	assert.Equal(t, mockStore, cacher.store)
	assert.Equal(t, time.Hour, cacher.defaultTTL)
}

func TestNewCacherWithTTL(t *testing.T) {
	mockStore := &MockStore{}
	customTTL := time.Minute * 30
	cacher := NewCacherWithTTL(mockStore, customTTL)
	
	assert.NotNil(t, cacher)
	assert.Equal(t, mockStore, cacher.store)
	assert.Equal(t, customTTL, cacher.defaultTTL)
}

func TestDefaultCacher_Get_CacheHit(t *testing.T) {
	mockStore := &MockStore{}
	cacher := NewCacher(mockStore)
	ctx := context.Background()

	// Setup mock for cache hit
	mockStore.On("Get", ctx, "test_key", mock.AnythingOfType("*string")).
		Run(func(args mock.Arguments) {
			dst := args.Get(2).(*string)
			*dst = "cached_value"
		}).
		Return(true, nil)

	var result string
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		t.Error("Fallback should not be called on cache hit")
		return nil, false, nil
	}

	found, err := cacher.Get(ctx, "test_key", &result, fallback, nil)

	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "cached_value", result)
	mockStore.AssertExpectations(t)
}

func TestDefaultCacher_Get_CacheMissWithFallback(t *testing.T) {
	mockStore := &MockStore{}
	cacher := NewCacher(mockStore)
	ctx := context.Background()

	// Setup mock for cache miss
	mockStore.On("Get", ctx, "test_key", mock.AnythingOfType("*string")).
		Return(false, nil)

	// Setup mock for MSet after fallback
	mockStore.On("MSet", ctx, mock.MatchedBy(func(items map[string]interface{}) bool {
		return items["test_key"] == "fallback_value"
	}), time.Hour).Return(nil)

	var result string
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		assert.Equal(t, "test_key", key)
		return "fallback_value", true, nil
	}

	found, err := cacher.Get(ctx, "test_key", &result, fallback, nil)

	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "fallback_value", result)
	mockStore.AssertExpectations(t)
}

func TestDefaultCacher_Get_CacheMissNoFallback(t *testing.T) {
	mockStore := &MockStore{}
	cacher := NewCacher(mockStore)
	ctx := context.Background()

	// Setup mock for cache miss
	mockStore.On("Get", ctx, "test_key", mock.AnythingOfType("*string")).
		Return(false, nil)

	var result string
	found, err := cacher.Get(ctx, "test_key", &result, nil, nil)

	assert.NoError(t, err)
	assert.False(t, found)
	mockStore.AssertExpectations(t)
}

func TestDefaultCacher_Get_FallbackNotFound(t *testing.T) {
	mockStore := &MockStore{}
	cacher := NewCacher(mockStore)
	ctx := context.Background()

	// Setup mock for cache miss
	mockStore.On("Get", ctx, "test_key", mock.AnythingOfType("*string")).
		Return(false, nil)

	var result string
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		return nil, false, nil // Not found in fallback either
	}

	found, err := cacher.Get(ctx, "test_key", &result, fallback, nil)

	assert.NoError(t, err)
	assert.False(t, found)
	mockStore.AssertExpectations(t)
}

func TestDefaultCacher_Get_FallbackError(t *testing.T) {
	mockStore := &MockStore{}
	cacher := NewCacher(mockStore)
	ctx := context.Background()

	// Setup mock for cache miss
	mockStore.On("Get", ctx, "test_key", mock.AnythingOfType("*string")).
		Return(false, nil)

	var result string
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		return nil, false, errors.New("fallback error")
	}

	found, err := cacher.Get(ctx, "test_key", &result, fallback, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fallback function failed")
	assert.False(t, found)
	mockStore.AssertExpectations(t)
}

func TestDefaultCacher_Get_CustomTTL(t *testing.T) {
	mockStore := &MockStore{}
	cacher := NewCacher(mockStore)
	ctx := context.Background()

	// Setup mock for cache miss
	mockStore.On("Get", ctx, "test_key", mock.AnythingOfType("*string")).
		Return(false, nil)

	// Setup mock for MSet with custom TTL
	customTTL := time.Minute * 30
	mockStore.On("MSet", ctx, mock.MatchedBy(func(items map[string]interface{}) bool {
		return items["test_key"] == "fallback_value"
	}), customTTL).Return(nil)

	var result string
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		return "fallback_value", true, nil
	}

	opts := &CacheOptions{TTL: customTTL}
	found, err := cacher.Get(ctx, "test_key", &result, fallback, opts)

	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "fallback_value", result)
	mockStore.AssertExpectations(t)
}

func TestDefaultCacher_MGet(t *testing.T) {
	mockStore := &MockStore{}
	cacher := NewCacher(mockStore)
	ctx := context.Background()

	keys := []string{"key1", "key2", "key3"}
	resultMap := make(map[string]string)

	// Setup mock for MGet - returns partial results
	mockStore.On("MGet", ctx, keys, &resultMap).
		Run(func(args mock.Arguments) {
			dstMap := args.Get(2).(*map[string]string)
			// Simulate cached value as serialized JSON string (without extra quotes)
			(*dstMap)["key1"] = "cached_value1"
			// key2 and key3 are missing (cache miss)
		}).
		Return(nil)

	// Setup mock for MSet after fallback
	mockStore.On("MSet", ctx, mock.MatchedBy(func(items map[string]interface{}) bool {
		return len(items) == 2 && items["key2"] == "fallback_value2" && items["key3"] == "fallback_value3"
	}), time.Hour).Return(nil)

	fallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		result := make(map[string]interface{})
		for _, key := range keys {
			if key == "key2" {
				result[key] = "fallback_value2"
			} else if key == "key3" {
				result[key] = "fallback_value3"
			}
		}
		return result, nil
	}

	err := cacher.MGet(ctx, keys, &resultMap, fallback, nil)

	assert.NoError(t, err)
	assert.Equal(t, "cached_value1", resultMap["key1"])
	assert.Equal(t, "fallback_value2", resultMap["key2"])
	assert.Equal(t, "fallback_value3", resultMap["key3"])
	mockStore.AssertExpectations(t)
}

func TestDefaultCacher_MGet_NoFallback(t *testing.T) {
	mockStore := &MockStore{}
	cacher := NewCacher(mockStore)
	ctx := context.Background()

	keys := []string{"key1", "key2"}
	resultMap := make(map[string]string)

	// Setup mock for MGet
	mockStore.On("MGet", ctx, keys, &resultMap).Return(nil)

	err := cacher.MGet(ctx, keys, &resultMap, nil, nil)

	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestDefaultCacher_MDelete(t *testing.T) {
	mockStore := &MockStore{}
	cacher := NewCacher(mockStore)
	ctx := context.Background()

	keys := []string{"key1", "key2", "key3"}

	// Setup mock for Del
	mockStore.On("Del", ctx, keys).Return(int64(2), nil)

	deleted, err := cacher.MDelete(ctx, keys)

	assert.NoError(t, err)
	assert.Equal(t, int64(2), deleted)
	mockStore.AssertExpectations(t)
}

func TestDefaultCacher_MRefresh(t *testing.T) {
	mockStore := &MockStore{}
	cacher := NewCacher(mockStore)
	ctx := context.Background()

	keys := []string{"key1", "key2"}
	resultMap := make(map[string]string)

	// Setup mock for Del (clear existing cache)
	mockStore.On("Del", ctx, keys).Return(int64(2), nil)

	// Setup mock for MSet (cache new data)
	mockStore.On("MSet", ctx, mock.MatchedBy(func(items map[string]interface{}) bool {
		return len(items) == 2 && items["key1"] == "new_value1" && items["key2"] == "new_value2"
	}), time.Hour).Return(nil)

	fallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		result := make(map[string]interface{})
		for _, key := range keys {
			if key == "key1" {
				result[key] = "new_value1"
			} else if key == "key2" {
				result[key] = "new_value2"
			}
		}
		return result, nil
	}

	err := cacher.MRefresh(ctx, keys, &resultMap, fallback, nil)

	assert.NoError(t, err)
	assert.Equal(t, "new_value1", resultMap["key1"])
	assert.Equal(t, "new_value2", resultMap["key2"])
	mockStore.AssertExpectations(t)
}

func TestDefaultCacher_MRefresh_NoFallback(t *testing.T) {
	mockStore := &MockStore{}
	cacher := NewCacher(mockStore)
	ctx := context.Background()

	keys := []string{"key1", "key2"}
	resultMap := make(map[string]string)

	err := cacher.MRefresh(ctx, keys, &resultMap, nil, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fallback function is required for refresh")
}

func TestDefaultCacher_Integration(t *testing.T) {
	// Use a real in-memory store for integration testing
	store := NewMemoryStore()
	cacher := NewCacher(store)
	ctx := context.Background()

	// Test complete workflow
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		return "fallback_value_" + key, true, nil
	}

	// First call should use fallback
	var result1 string
	found, err := cacher.Get(ctx, "integration_key", &result1, fallback, nil)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "fallback_value_integration_key", result1)

	// Second call should use cache
	var result2 string
	found, err = cacher.Get(ctx, "integration_key", &result2, nil, nil)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "fallback_value_integration_key", result2)

	// Delete and verify
	deleted, err := cacher.MDelete(ctx, []string{"integration_key"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	// Should not find after delete
	var result3 string
	found, err = cacher.Get(ctx, "integration_key", &result3, nil, nil)
	require.NoError(t, err)
	assert.False(t, found)
}

// Simple in-memory store for integration testing
type MemoryStore struct {
	data map[string]string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data: make(map[string]string),
	}
}

func (m *MemoryStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	value, exists := m.data[key]
	if !exists {
		return false, nil
	}
	return true, DeserializeValue(value, dst)
}

func (m *MemoryStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	if err := ValidateMapPointer(dstMap); err != nil {
		return err
	}
	
	for _, key := range keys {
		if value, exists := m.data[key]; exists {
			if err := SetMapValue(dstMap, key, value); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *MemoryStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool)
	for _, key := range keys {
		_, exists := m.data[key]
		result[key] = exists
	}
	return result, nil
}

func (m *MemoryStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	for key, value := range items {
		serialized, err := SerializeValue(value)
		if err != nil {
			return err
		}
		m.data[key] = serialized
	}
	return nil
}

func (m *MemoryStore) Del(ctx context.Context, keys ...string) (int64, error) {
	var deleted int64
	for _, key := range keys {
		if _, exists := m.data[key]; exists {
			delete(m.data, key)
			deleted++
		}
	}
	return deleted, nil
}