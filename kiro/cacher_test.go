package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockStore 模拟Store实现
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

func TestCacher_Get_CacheHit(t *testing.T) {
	mockStore := new(MockStore)
	cacher := NewCacher(mockStore)
	ctx := context.Background()

	// 模拟缓存命中
	mockStore.On("Get", ctx, "key1", mock.AnythingOfType("*string")).Run(func(args mock.Arguments) {
		dst := args.Get(2).(*string)
		*dst = "cached_value"
	}).Return(true, nil)

	var result string
	found, err := cacher.Get(ctx, "key1", &result, nil, nil)

	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "cached_value", result)
	mockStore.AssertExpectations(t)
}

func TestCacher_Get_CacheMissWithFallback(t *testing.T) {
	mockStore := new(MockStore)
	cacher := NewCacher(mockStore)
	ctx := context.Background()

	// 模拟缓存未命中
	mockStore.On("Get", ctx, "key1", mock.AnythingOfType("*string")).Return(false, nil)
	
	// 模拟缓存写入
	mockStore.On("MSet", ctx, map[string]interface{}{"key1": "fallback_value"}, time.Duration(0)).Return(nil)

	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		return "fallback_value", true, nil
	}

	var result string
	found, err := cacher.Get(ctx, "key1", &result, fallback, nil)

	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "fallback_value", result)
	mockStore.AssertExpectations(t)
}

func TestCacher_Get_CacheMissNoFallback(t *testing.T) {
	mockStore := new(MockStore)
	cacher := NewCacher(mockStore)
	ctx := context.Background()

	// 模拟缓存未命中
	mockStore.On("Get", ctx, "key1", mock.AnythingOfType("*string")).Return(false, nil)

	var result string
	found, err := cacher.Get(ctx, "key1", &result, nil, nil)

	assert.NoError(t, err)
	assert.False(t, found)
	assert.Equal(t, "", result)
	mockStore.AssertExpectations(t)
}

func TestCacher_Get_FallbackError(t *testing.T) {
	mockStore := new(MockStore)
	cacher := NewCacher(mockStore)
	ctx := context.Background()

	// 模拟缓存未命中
	mockStore.On("Get", ctx, "key1", mock.AnythingOfType("*string")).Return(false, nil)

	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		return nil, false, errors.New("fallback error")
	}

	var result string
	found, err := cacher.Get(ctx, "key1", &result, fallback, nil)

	assert.Error(t, err)
	assert.False(t, found)
	assert.Equal(t, "fallback error", err.Error())
	mockStore.AssertExpectations(t)
}

func TestCacher_Get_WithTTL(t *testing.T) {
	mockStore := new(MockStore)
	cacher := NewCacher(mockStore)
	ctx := context.Background()

	// 模拟缓存未命中
	mockStore.On("Get", ctx, "key1", mock.AnythingOfType("*string")).Return(false, nil)
	
	// 模拟缓存写入，带TTL
	ttl := 5 * time.Minute
	mockStore.On("MSet", ctx, map[string]interface{}{"key1": "fallback_value"}, ttl).Return(nil)

	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		return "fallback_value", true, nil
	}

	opts := &CacheOptions{TTL: ttl}
	var result string
	found, err := cacher.Get(ctx, "key1", &result, fallback, opts)

	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "fallback_value", result)
	mockStore.AssertExpectations(t)
}

func TestCacher_MDelete(t *testing.T) {
	mockStore := new(MockStore)
	cacher := NewCacher(mockStore)
	ctx := context.Background()

	keys := []string{"key1", "key2", "key3"}
	mockStore.On("Del", ctx, keys).Return(int64(2), nil)

	deleted, err := cacher.MDelete(ctx, keys)

	assert.NoError(t, err)
	assert.Equal(t, int64(2), deleted)
	mockStore.AssertExpectations(t)
}

func TestCacher_MDelete_EmptyKeys(t *testing.T) {
	mockStore := new(MockStore)
	cacher := NewCacher(mockStore)
	ctx := context.Background()

	deleted, err := cacher.MDelete(ctx, []string{})

	assert.NoError(t, err)
	assert.Equal(t, int64(0), deleted)
	mockStore.AssertNotCalled(t, "Del")
}