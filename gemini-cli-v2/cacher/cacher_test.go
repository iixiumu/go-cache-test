package cacher

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockStore is a mock of Store interface
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

func TestCacher_Get(t *testing.T) {
	ctx := context.Background()

	t.Run("cache hit", func(t *testing.T) {
		store := new(MockStore)
		cacher := NewCacher(store)
		key := "test_key"
		var dst string

		store.On("Get", ctx, key, &dst).Return(true, nil).Run(func(args mock.Arguments) {
			arg := args.Get(2).(*string)
			*arg = "cached_value"
		})

		found, err := cacher.Get(ctx, key, &dst, nil, nil)

		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "cached_value", dst)
		store.AssertExpectations(t)
	})

	t.Run("cache miss, fallback success", func(t *testing.T) {
		store := new(MockStore)
		cacher := NewCacher(store)
		key := "test_key"
		var dst string
		fallbackValue := "fallback_value"

		store.On("Get", ctx, key, &dst).Return(false, nil)
		store.On("MSet", ctx, map[string]interface{}{key: fallbackValue}, time.Duration(0)).Return(nil)

		fallback := func(ctx context.Context, k string) (interface{}, bool, error) {
			assert.Equal(t, key, k)
			return fallbackValue, true, nil
		}

		found, err := cacher.Get(ctx, key, &dst, fallback, nil)

		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, fallbackValue, dst)
		store.AssertExpectations(t)
	})

	t.Run("cache miss, fallback not found", func(t *testing.T) {
		store := new(MockStore)
		cacher := NewCacher(store)
		key := "test_key"
		var dst string

		store.On("Get", ctx, key, &dst).Return(false, nil)

		fallback := func(ctx context.Context, k string) (interface{}, bool, error) {
			return nil, false, nil
		}

		found, err := cacher.Get(ctx, key, &dst, fallback, nil)

		assert.NoError(t, err)
		assert.False(t, found)
		store.AssertExpectations(t)
	})

	t.Run("cache miss, fallback error", func(t *testing.T) {
		store := new(MockStore)
		cacher := NewCacher(store)
		key := "test_key"
		var dst string
		fallbackErr := errors.New("fallback error")

		store.On("Get", ctx, key, &dst).Return(false, nil)

		fallback := func(ctx context.Context, k string) (interface{}, bool, error) {
			return nil, false, fallbackErr
		}

		found, err := cacher.Get(ctx, key, &dst, fallback, nil)

		assert.Error(t, err)
		assert.Equal(t, fallbackErr, err)
		assert.False(t, found)
		store.AssertExpectations(t)
	})
}

func TestCacher_MGet(t *testing.T) {
	ctx := context.Background()

	t.Run("all keys hit", func(t *testing.T) {
		store := new(MockStore)
		cacher := NewCacher(store)
		keys := []string{"key1", "key2"}
		dst := make(map[string]string)

		store.On("MGet", ctx, keys, &dst).Return(nil).Run(func(args mock.Arguments) {
			arg := args.Get(2).(*map[string]string)
			(*arg)["key1"] = "value1"
			(*arg)["key2"] = "value2"
		})

		err := cacher.MGet(ctx, keys, &dst, nil, nil)

		assert.NoError(t, err)
		assert.Equal(t, "value1", dst["key1"])
		assert.Equal(t, "value2", dst["key2"])
		store.AssertExpectations(t)
	})
}

func TestCacher_MDelete(t *testing.T) {
	ctx := context.Background()
	store := new(MockStore)
	cacher := NewCacher(store)
	keys := []string{"key1", "key2"}

	store.On("Del", ctx, keys).Return(int64(2), nil)

	deleted, err := cacher.MDelete(ctx, keys)

	assert.NoError(t, err)
	assert.Equal(t, int64(2), deleted)
	store.AssertExpectations(t)
}

func TestCacher_MRefresh(t *testing.T) {
	ctx := context.Background()
	store := new(MockStore)
	cacher := NewCacher(store)
	keys := []string{"key1", "key2"}
	dst := make(map[string]string)
	fallbackData := map[string]interface{}{"key1": "new_value1", "key2": "new_value2"}

	store.On("MSet", ctx, fallbackData, time.Duration(0)).Return(nil)

	fallback := func(ctx context.Context, k []string) (map[string]interface{}, error) {
		assert.Equal(t, keys, k)
		return fallbackData, nil
	}

	err := cacher.MRefresh(ctx, keys, &dst, fallback, nil)

	assert.NoError(t, err)
	assert.Equal(t, "new_value1", dst["key1"])
	assert.Equal(t, "new_value2", dst["key2"])
	store.AssertExpectations(t)
}
