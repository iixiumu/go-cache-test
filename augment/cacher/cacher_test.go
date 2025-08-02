package cacher

import (
	"context"
	"errors"
	"testing"
	"time"

	cache "go-cache"

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

func TestDefaultCacher_Get(t *testing.T) {
	mockStore := new(MockStore)
	cacher := NewDefaultCacher(mockStore, time.Hour)
	ctx := context.Background()

	t.Run("CacheHit", func(t *testing.T) {
		// 模拟缓存命中
		mockStore.On("Get", ctx, "test_key", mock.AnythingOfType("*string")).Return(true, nil).Run(func(args mock.Arguments) {
			dst := args.Get(2).(*string)
			*dst = "cached_value"
		})

		var result string
		found, err := cacher.Get(ctx, "test_key", &result, nil, nil)

		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "cached_value", result)
		mockStore.AssertExpectations(t)
	})

	t.Run("CacheMissWithFallback", func(t *testing.T) {
		mockStore.ExpectedCalls = nil // 清除之前的期望

		// 模拟缓存未命中
		mockStore.On("Get", ctx, "miss_key", mock.AnythingOfType("*string")).Return(false, nil)

		// 模拟缓存设置
		mockStore.On("MSet", ctx, mock.MatchedBy(func(items map[string]interface{}) bool {
			return items["miss_key"] == "fallback_value"
		}), time.Hour).Return(nil)

		// 定义回退函数
		fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
			assert.Equal(t, "miss_key", key)
			return "fallback_value", true, nil
		}

		var result string
		found, err := cacher.Get(ctx, "miss_key", &result, fallback, nil)

		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "fallback_value", result)
		mockStore.AssertExpectations(t)
	})

	t.Run("CacheMissNoFallback", func(t *testing.T) {
		mockStore.ExpectedCalls = nil

		// 模拟缓存未命中
		mockStore.On("Get", ctx, "no_fallback_key", mock.AnythingOfType("*string")).Return(false, nil)

		var result string
		found, err := cacher.Get(ctx, "no_fallback_key", &result, nil, nil)

		assert.NoError(t, err)
		assert.False(t, found)
		mockStore.AssertExpectations(t)
	})

	t.Run("FallbackNotFound", func(t *testing.T) {
		mockStore.ExpectedCalls = nil

		// 模拟缓存未命中
		mockStore.On("Get", ctx, "not_found_key", mock.AnythingOfType("*string")).Return(false, nil)

		// 定义回退函数返回未找到
		fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
			return nil, false, nil
		}

		var result string
		found, err := cacher.Get(ctx, "not_found_key", &result, fallback, nil)

		assert.NoError(t, err)
		assert.False(t, found)
		mockStore.AssertExpectations(t)
	})

	t.Run("FallbackError", func(t *testing.T) {
		mockStore.ExpectedCalls = nil

		// 模拟缓存未命中
		mockStore.On("Get", ctx, "error_key", mock.AnythingOfType("*string")).Return(false, nil)

		// 定义回退函数返回错误
		fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
			return nil, false, errors.New("fallback error")
		}

		var result string
		found, err := cacher.Get(ctx, "error_key", &result, fallback, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "fallback function failed")
		assert.False(t, found)
		mockStore.AssertExpectations(t)
	})

	t.Run("InvalidDestination", func(t *testing.T) {
		var result string
		found, err := cacher.Get(ctx, "test_key", result, nil, nil) // 传递非指针

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "pointer")
		assert.False(t, found)
	})

	t.Run("CustomTTL", func(t *testing.T) {
		mockStore.ExpectedCalls = nil

		// 模拟缓存未命中
		mockStore.On("Get", ctx, "ttl_key", mock.AnythingOfType("*string")).Return(false, nil)

		// 模拟缓存设置，验证使用了自定义TTL
		customTTL := 30 * time.Minute
		mockStore.On("MSet", ctx, mock.MatchedBy(func(items map[string]interface{}) bool {
			return items["ttl_key"] == "ttl_value"
		}), customTTL).Return(nil)

		// 定义回退函数
		fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
			return "ttl_value", true, nil
		}

		opts := &cache.CacheOptions{TTL: customTTL}
		var result string
		found, err := cacher.Get(ctx, "ttl_key", &result, fallback, opts)

		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "ttl_value", result)
		mockStore.AssertExpectations(t)
	})
}

func TestDefaultCacher_MGet(t *testing.T) {
	mockStore := new(MockStore)
	cacher := NewDefaultCacher(mockStore, time.Hour)
	ctx := context.Background()

	t.Run("PartialCacheHit", func(t *testing.T) {
		keys := []string{"key1", "key2", "key3"}

		// 模拟部分缓存命中
		mockStore.On("MGet", ctx, keys, mock.AnythingOfType("*map[string]string")).Return(nil).Run(func(args mock.Arguments) {
			dstMap := args.Get(2).(*map[string]string)
			*dstMap = map[string]string{
				"key1": "cached_value1",
				"key3": "cached_value3",
			}
		})

		// 模拟批量回退
		mockStore.On("MSet", ctx, mock.MatchedBy(func(items map[string]interface{}) bool {
			return items["key2"] == "fallback_value2"
		}), time.Hour).Return(nil)

		// 定义批量回退函数
		fallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
			assert.Equal(t, []string{"key2"}, keys) // 只有key2未命中
			return map[string]interface{}{
				"key2": "fallback_value2",
			}, nil
		}

		result := make(map[string]string)
		err := cacher.MGet(ctx, keys, &result, fallback, nil)

		assert.NoError(t, err)
		assert.Equal(t, "cached_value1", result["key1"])
		assert.Equal(t, "fallback_value2", result["key2"])
		assert.Equal(t, "cached_value3", result["key3"])
		mockStore.AssertExpectations(t)
	})

	t.Run("AllCacheHit", func(t *testing.T) {
		mockStore.ExpectedCalls = nil
		keys := []string{"key1", "key2"}

		// 模拟全部缓存命中
		mockStore.On("MGet", ctx, keys, mock.AnythingOfType("*map[string]string")).Return(nil).Run(func(args mock.Arguments) {
			dstMap := args.Get(2).(*map[string]string)
			*dstMap = map[string]string{
				"key1": "cached_value1",
				"key2": "cached_value2",
			}
		})

		result := make(map[string]string)
		err := cacher.MGet(ctx, keys, &result, nil, nil)

		assert.NoError(t, err)
		assert.Equal(t, "cached_value1", result["key1"])
		assert.Equal(t, "cached_value2", result["key2"])
		mockStore.AssertExpectations(t)
	})

	t.Run("AllCacheMiss", func(t *testing.T) {
		mockStore.ExpectedCalls = nil
		keys := []string{"key1", "key2"}

		// 模拟全部缓存未命中
		mockStore.On("MGet", ctx, keys, mock.AnythingOfType("*map[string]string")).Return(nil).Run(func(args mock.Arguments) {
			dstMap := args.Get(2).(*map[string]string)
			*dstMap = make(map[string]string)
		})

		// 模拟批量回退
		mockStore.On("MSet", ctx, mock.MatchedBy(func(items map[string]interface{}) bool {
			return len(items) == 2 && items["key1"] == "fallback_value1" && items["key2"] == "fallback_value2"
		}), time.Hour).Return(nil)

		// 定义批量回退函数
		fallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
			assert.ElementsMatch(t, []string{"key1", "key2"}, keys)
			return map[string]interface{}{
				"key1": "fallback_value1",
				"key2": "fallback_value2",
			}, nil
		}

		result := make(map[string]string)
		err := cacher.MGet(ctx, keys, &result, fallback, nil)

		assert.NoError(t, err)
		assert.Equal(t, "fallback_value1", result["key1"])
		assert.Equal(t, "fallback_value2", result["key2"])
		mockStore.AssertExpectations(t)
	})

	t.Run("EmptyKeys", func(t *testing.T) {
		mockStore.ExpectedCalls = nil

		// 模拟空键列表
		mockStore.On("MGet", ctx, []string{}, mock.AnythingOfType("*map[string]string")).Return(nil)

		result := make(map[string]string)
		err := cacher.MGet(ctx, []string{}, &result, nil, nil)

		assert.NoError(t, err)
		assert.Empty(t, result)
		mockStore.AssertExpectations(t)
	})

	t.Run("InvalidDestination", func(t *testing.T) {
		keys := []string{"key1"}
		var result string // 非map指针

		err := cacher.MGet(ctx, keys, &result, nil, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "map")
	})

	t.Run("FallbackError", func(t *testing.T) {
		mockStore.ExpectedCalls = nil
		keys := []string{"key1"}

		// 模拟缓存未命中
		mockStore.On("MGet", ctx, keys, mock.AnythingOfType("*map[string]string")).Return(nil).Run(func(args mock.Arguments) {
			dstMap := args.Get(2).(*map[string]string)
			*dstMap = make(map[string]string)
		})

		// 定义回退函数返回错误
		fallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
			return nil, errors.New("fallback error")
		}

		result := make(map[string]string)
		err := cacher.MGet(ctx, keys, &result, fallback, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "batch fallback function failed")
		mockStore.AssertExpectations(t)
	})
}

func TestDefaultCacher_MDelete(t *testing.T) {
	mockStore := new(MockStore)
	cacher := NewDefaultCacher(mockStore, time.Hour)
	ctx := context.Background()

	t.Run("DeleteKeys", func(t *testing.T) {
		keys := []string{"key1", "key2", "key3"}

		mockStore.On("Del", ctx, keys).Return(int64(2), nil)

		deleted, err := cacher.MDelete(ctx, keys)

		assert.NoError(t, err)
		assert.Equal(t, int64(2), deleted)
		mockStore.AssertExpectations(t)
	})

	t.Run("DeleteError", func(t *testing.T) {
		keys := []string{"key1"}

		mockStore.On("Del", ctx, keys).Return(int64(0), errors.New("delete error"))

		deleted, err := cacher.MDelete(ctx, keys)

		assert.Error(t, err)
		assert.Equal(t, int64(0), deleted)
		mockStore.AssertExpectations(t)
	})
}

func TestDefaultCacher_MRefresh(t *testing.T) {
	mockStore := new(MockStore)
	cacher := NewDefaultCacher(mockStore, time.Hour)
	ctx := context.Background()

	t.Run("RefreshSuccess", func(t *testing.T) {
		keys := []string{"key1", "key2"}

		// 模拟删除现有缓存
		mockStore.On("Del", ctx, keys).Return(int64(2), nil)

		// 模拟设置新缓存
		mockStore.On("MSet", ctx, mock.MatchedBy(func(items map[string]interface{}) bool {
			return len(items) == 2 && items["key1"] == "new_value1" && items["key2"] == "new_value2"
		}), time.Hour).Return(nil)

		// 定义批量回退函数
		fallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
			assert.ElementsMatch(t, []string{"key1", "key2"}, keys)
			return map[string]interface{}{
				"key1": "new_value1",
				"key2": "new_value2",
			}, nil
		}

		result := make(map[string]string)
		err := cacher.MRefresh(ctx, keys, &result, fallback, nil)

		assert.NoError(t, err)
		assert.Equal(t, "new_value1", result["key1"])
		assert.Equal(t, "new_value2", result["key2"])
		mockStore.AssertExpectations(t)
	})

	t.Run("RefreshWithoutFallback", func(t *testing.T) {
		mockStore.ExpectedCalls = nil
		keys := []string{"key1"}

		// 模拟删除现有缓存
		mockStore.On("Del", ctx, keys).Return(int64(1), nil)

		result := make(map[string]string)
		err := cacher.MRefresh(ctx, keys, &result, nil, nil)

		assert.NoError(t, err)
		assert.Empty(t, result)
		mockStore.AssertExpectations(t)
	})

	t.Run("RefreshDeleteError", func(t *testing.T) {
		mockStore.ExpectedCalls = nil
		keys := []string{"key1"}

		// 模拟删除失败
		mockStore.On("Del", ctx, keys).Return(int64(0), errors.New("delete error"))

		result := make(map[string]string)
		err := cacher.MRefresh(ctx, keys, &result, nil, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete keys for refresh")
		mockStore.AssertExpectations(t)
	})

	t.Run("RefreshFallbackError", func(t *testing.T) {
		mockStore.ExpectedCalls = nil
		keys := []string{"key1"}

		// 模拟删除成功
		mockStore.On("Del", ctx, keys).Return(int64(1), nil)

		// 定义回退函数返回错误
		fallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
			return nil, errors.New("fallback error")
		}

		result := make(map[string]string)
		err := cacher.MRefresh(ctx, keys, &result, fallback, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "batch fallback function failed during refresh")
		mockStore.AssertExpectations(t)
	})

	t.Run("RefreshCacheSetError", func(t *testing.T) {
		mockStore.ExpectedCalls = nil
		keys := []string{"key1"}

		// 模拟删除成功
		mockStore.On("Del", ctx, keys).Return(int64(1), nil)

		// 模拟缓存设置失败
		mockStore.On("MSet", ctx, mock.MatchedBy(func(items map[string]interface{}) bool {
			return items["key1"] == "new_value1"
		}), time.Hour).Return(errors.New("cache set error"))

		// 定义批量回退函数
		fallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
			return map[string]interface{}{
				"key1": "new_value1",
			}, nil
		}

		result := make(map[string]string)
		err := cacher.MRefresh(ctx, keys, &result, fallback, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to cache refreshed data")
		mockStore.AssertExpectations(t)
	})

	t.Run("InvalidDestination", func(t *testing.T) {
		keys := []string{"key1"}
		var result string // 非map指针

		err := cacher.MRefresh(ctx, keys, &result, nil, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "map")
	})
}
