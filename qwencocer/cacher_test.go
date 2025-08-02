package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
	return int64(args.Int(0)), args.Error(1)
}

func TestCacher_Get(t *testing.T) {
	// Create a mock store
	mockStore := new(MockStore)
	cacher := NewCacher(mockStore)

	// Test case 1: Cache hit
	t.Run("CacheHit", func(t *testing.T) {
		var result string
		mockStore.On("Get", mock.Anything, "key1", mock.AnythingOfType("*string")).Return(true, nil).Run(func(args mock.Arguments) {
			dst := args.Get(2).(*string)
			*dst = "cached_value"
		})

		found, err := cacher.Get(context.Background(), "key1", &result, nil, nil)
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "cached_value", result)

		mockStore.AssertExpectations(t)
	})

	// Test case 2: Cache miss with fallback
	t.Run("CacheMissWithFallback", func(t *testing.T) {
		var result string
		// First call to Get returns false (cache miss)
		mockStore.On("Get", mock.Anything, "key2", mock.AnythingOfType("*string")).Return(false, nil).Once()
		// MSet is called to cache the fallback value
		mockStore.On("MSet", mock.Anything, map[string]interface{}{"key2": "fallback_value"}, DefaultTTL).Return(nil).Once()

		fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
			return "fallback_value", true, nil
		}

		found, err := cacher.Get(context.Background(), "key2", &result, fallback, nil)
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "fallback_value", result)

		mockStore.AssertExpectations(t)
	})

	// Test case 3: Cache miss without fallback
	t.Run("CacheMissWithoutFallback", func(t *testing.T) {
		var result string
		mockStore.On("Get", mock.Anything, "key3", mock.AnythingOfType("*string")).Return(false, nil).Once()

		found, err := cacher.Get(context.Background(), "key3", &result, nil, nil)
		assert.NoError(t, err)
		assert.False(t, found)
		assert.Equal(t, "", result) // result should be zero value

		mockStore.AssertExpectations(t)
	})

	// Test case 4: Cache error
	t.Run("CacheError", func(t *testing.T) {
		var result string
		mockStore.On("Get", mock.Anything, "key4", mock.AnythingOfType("*string")).Return(false, assert.AnError).Once()

		found, err := cacher.Get(context.Background(), "key4", &result, nil, nil)
		assert.Error(t, err)
		assert.False(t, found)

		mockStore.AssertExpectations(t)
	})

	// Test case 5: Fallback error
	t.Run("FallbackError", func(t *testing.T) {
		var result string
		mockStore.On("Get", mock.Anything, "key5", mock.AnythingOfType("*string")).Return(false, nil).Once()

		fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
			return nil, false, assert.AnError
		}

		found, err := cacher.Get(context.Background(), "key5", &result, fallback, nil)
		assert.Error(t, err)
		assert.False(t, found)

		mockStore.AssertExpectations(t)
	})

	// Test case 6: Invalid dst parameter
	t.Run("InvalidDstParameter", func(t *testing.T) {
		found, err := cacher.Get(context.Background(), "key6", "not_a_pointer", nil, nil)
		assert.Error(t, err)
		assert.False(t, found)
		assert.Contains(t, err.Error(), "dst must be a non-nil pointer")

		found, err = cacher.Get(context.Background(), "key7", (*string)(nil), nil, nil)
		assert.Error(t, err)
		assert.False(t, found)
		assert.Contains(t, err.Error(), "dst must be a non-nil pointer")
	})
}

func TestCacher_MGet(t *testing.T) {

	// Test case 1: All keys hit
	t.Run("AllKeysHit", func(t *testing.T) {
		// Create a mock store
		mockStore := new(MockStore)
		cacher := NewCacher(mockStore)
		result := make(map[string]string)
		expected := map[string]string{"key1": "value1", "key2": "value2"}

		mockStore.On("MGet", mock.Anything, []string{"key1", "key2"}, mock.AnythingOfType("*map[string]string")).Return(nil).Run(func(args mock.Arguments) {
			dstMap := args.Get(2).(*map[string]string)
			*dstMap = expected
		})

		err := cacher.MGet(context.Background(), []string{"key1", "key2"}, &result, nil, nil)
		assert.NoError(t, err)
		assert.Equal(t, expected, result)

		mockStore.AssertExpectations(t)
	})

	// Test case 2: Partial hit with fallback
	t.Run("PartialHitWithFallback", func(t *testing.T) {
		// Create a mock store
		mockStore := new(MockStore)
		cacher := NewCacher(mockStore)
		result := make(map[string]string)
		cached := map[string]string{"key1": "value1"}
		fallbackValues := map[string]interface{}{"key2": "value2"}

		// MGet returns partial results
		mockStore.On("MGet", mock.Anything, []string{"key1", "key2"}, mock.AnythingOfType("*map[string]string")).Return(nil).Run(func(args mock.Arguments) {
			dstMap := args.Get(2).(*map[string]string)
			*dstMap = cached
		})
		// MSet is called to cache the fallback values
		mockStore.On("MSet", mock.Anything, fallbackValues, DefaultTTL).Return(nil).Once()

		fallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
			return fallbackValues, nil
		}

		err := cacher.MGet(context.Background(), []string{"key1", "key2"}, &result, fallback, nil)
		assert.NoError(t, err)
		assert.Equal(t, "value1", result["key1"])
		assert.Equal(t, "value2", result["key2"])

		mockStore.AssertExpectations(t)
	})

	// Test case 3: All keys miss with fallback
	t.Run("AllKeysMissWithFallback", func(t *testing.T) {
		// Create a mock store
		mockStore := new(MockStore)
		cacher := NewCacher(mockStore)
		result := make(map[string]string)
		fallbackValues := map[string]interface{}{"key1": "value1", "key2": "value2"}

		// MGet returns empty results
		mockStore.On("MGet", mock.Anything, []string{"key1", "key2"}, mock.AnythingOfType("*map[string]string")).Return(nil).Run(func(args mock.Arguments) {
			dstMap := args.Get(2).(*map[string]string)
			*dstMap = make(map[string]string)
		})
		// MSet is called to cache the fallback values
		mockStore.On("MSet", mock.Anything, fallbackValues, DefaultTTL).Return(nil).Once()

		fallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
			return fallbackValues, nil
		}

		err := cacher.MGet(context.Background(), []string{"key1", "key2"}, &result, fallback, nil)
		assert.NoError(t, err)
		assert.Equal(t, "value1", result["key1"])
		assert.Equal(t, "value2", result["key2"])

		mockStore.AssertExpectations(t)
	})

	// Test case 4: All keys miss without fallback
	t.Run("AllKeysMissWithoutFallback", func(t *testing.T) {
		// Create a mock store
		mockStore := new(MockStore)
		cacher := NewCacher(mockStore)
		result := make(map[string]string)

		// MGet returns empty results
		mockStore.On("MGet", mock.Anything, []string{"key1", "key2"}, mock.AnythingOfType("*map[string]string")).Return(nil).Run(func(args mock.Arguments) {
			dstMap := args.Get(2).(*map[string]string)
			*dstMap = make(map[string]string)
		})

		err := cacher.MGet(context.Background(), []string{"key1", "key2"}, &result, nil, nil)
		assert.NoError(t, err)
		assert.Empty(t, result)

		mockStore.AssertExpectations(t)
	})

	// Test case 5: MGet error
	t.Run("MGetError", func(t *testing.T) {
		// Create a mock store
		mockStore := new(MockStore)
		cacher := NewCacher(mockStore)
		result := make(map[string]string)

		mockStore.On("MGet", mock.Anything, []string{"key1", "key2"}, mock.AnythingOfType("*map[string]string")).Return(assert.AnError)

		err := cacher.MGet(context.Background(), []string{"key1", "key2"}, &result, nil, nil)
		assert.Error(t, err)

		mockStore.AssertExpectations(t)
	})

	// Test case 6: Fallback error
	t.Run("FallbackError", func(t *testing.T) {
		// Create a mock store
		mockStore := new(MockStore)
		cacher := NewCacher(mockStore)
		result := make(map[string]string)

		// MGet returns empty results
		mockStore.On("MGet", mock.Anything, []string{"key1", "key2"}, mock.AnythingOfType("*map[string]string")).Return(nil).Run(func(args mock.Arguments) {
			dstMap := args.Get(2).(*map[string]string)
			*dstMap = make(map[string]string)
		})

		fallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
			return nil, assert.AnError
		}

		err := cacher.MGet(context.Background(), []string{"key1", "key2"}, &result, fallback, nil)
		assert.Error(t, err)

		mockStore.AssertExpectations(t)
	})

	// Test case 7: Invalid dstMap parameter
	t.Run("InvalidDstMapParameter", func(t *testing.T) {
		// Create a mock store
		mockStore := new(MockStore)
		cacher := NewCacher(mockStore)
		err := cacher.MGet(context.Background(), []string{"key1"}, "not_a_pointer", nil, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dstMap must be a non-nil pointer")

		err = cacher.MGet(context.Background(), []string{"key1"}, (*map[string]string)(nil), nil, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dstMap must be a non-nil pointer")

		var notAMap []string
		err = cacher.MGet(context.Background(), []string{"key1"}, &notAMap, nil, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dstMap must be a pointer to a map")
		
		mockStore.AssertExpectations(t)
	})
}

func TestCacher_MDelete(t *testing.T) {
	// Create a mock store
	mockStore := new(MockStore)
	cacher := NewCacher(mockStore)

	// Test case: Delete keys
	t.Run("DeleteKeys", func(t *testing.T) {
		mockStore.On("Del", mock.Anything, []string{"key1", "key2"}).Return(2, nil)

		count, err := cacher.MDelete(context.Background(), []string{"key1", "key2"})
		assert.NoError(t, err)
		assert.Equal(t, int64(2), count)

		mockStore.AssertExpectations(t)
	})
}

func TestCacher_MRefresh(t *testing.T) {
	// Create a mock store
	mockStore := new(MockStore)
	cacher := NewCacher(mockStore)

	// Test case 1: Refresh with fallback
	t.Run("RefreshWithFallback", func(t *testing.T) {
		result := make(map[string]string)
		refreshedValues := map[string]interface{}{"key1": "new_value1", "key2": "new_value2"}

		// Fallback function returns new values
		fallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
			return refreshedValues, nil
		}
		// MSet is called to cache the refreshed values
		mockStore.On("MSet", mock.Anything, refreshedValues, DefaultTTL).Return(nil).Once()

		err := cacher.MRefresh(context.Background(), []string{"key1", "key2"}, &result, fallback, nil)
		assert.NoError(t, err)
		assert.Equal(t, "new_value1", result["key1"])
		assert.Equal(t, "new_value2", result["key2"])

		mockStore.AssertExpectations(t)
	})

	// Test case 2: Refresh without fallback
	t.Run("RefreshWithoutFallback", func(t *testing.T) {
		result := make(map[string]string)

		err := cacher.MRefresh(context.Background(), []string{"key1", "key2"}, &result, nil, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "fallback cannot be nil for MRefresh")
	})

	// Test case 3: Fallback error
	t.Run("FallbackError", func(t *testing.T) {
		result := make(map[string]string)

		fallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
			return nil, assert.AnError
		}

		err := cacher.MRefresh(context.Background(), []string{"key1", "key2"}, &result, fallback, nil)
		assert.Error(t, err)

		mockStore.AssertExpectations(t)
	})

	// Test case 4: Invalid dstMap parameter
	t.Run("InvalidDstMapParameter", func(t *testing.T) {
		fallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
			return map[string]interface{}{}, nil
		}

		err := cacher.MRefresh(context.Background(), []string{"key1"}, "not_a_pointer", fallback, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dstMap must be a non-nil pointer")

		err = cacher.MRefresh(context.Background(), []string{"key1"}, (*map[string]string)(nil), fallback, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dstMap must be a non-nil pointer")

		var notAMap []string
		err = cacher.MRefresh(context.Background(), []string{"key1"}, &notAMap, fallback, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dstMap must be a pointer to a map")
	})
}