package cacher

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockStore 是 Store 接口的模拟实现
type mockStore struct {
	mock.Mock
	data map[string]interface{}
}

func newMockStore() *mockStore {
	return &mockStore{
		data: make(map[string]interface{}),
	}
}

func (m *mockStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	args := m.Called(ctx, key, dst)
	
	// 如果找到键，则将值复制到dst
	if args.Bool(0) && m.data[key] != nil {
		// 使用反射设置dst的值
		dstValue := reflect.ValueOf(dst).Elem()
		value := reflect.ValueOf(m.data[key])
		if value.Type().AssignableTo(dstValue.Type()) {
			dstValue.Set(value)
		} else if value.Type().ConvertibleTo(dstValue.Type()) {
			dstValue.Set(value.Convert(dstValue.Type()))
		}
	}
	
	return args.Bool(0), args.Error(1)
}

func (m *mockStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	args := m.Called(ctx, keys, dstMap)
	
	// 如果有模拟数据，填充到dstMap
	if len(args) > 0 && args.Get(0) != nil {
		srcMap := args.Get(0).(map[string]interface{})
		dstMapValue := reflect.ValueOf(dstMap).Elem()
		
		for k, v := range srcMap {
			keyValue := reflect.ValueOf(k)
			valueValue := reflect.ValueOf(v)
			dstMapValue.SetMapIndex(keyValue, valueValue)
		}
		return args.Error(1)
	}
	
	return args.Error(0)
}

func (m *mockStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	args := m.Called(ctx, keys)
	return args.Get(0).(map[string]bool), args.Error(1)
}

func (m *mockStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	args := m.Called(ctx, items, ttl)
	
	// 保存数据
	for k, v := range items {
		m.data[k] = v
	}
	
	return args.Error(0)
}

func (m *mockStore) Del(ctx context.Context, keys ...string) (int64, error) {
	args := m.Called(ctx, keys)
	
	// 删除数据
	var count int64
	for _, key := range keys {
		if _, exists := m.data[key]; exists {
			delete(m.data, key)
			count++
		}
	}
	
	return count, args.Error(1)
}

func TestCacher_Get(t *testing.T) {
	ctx := context.Background()

	t.Run("cache hit", func(t *testing.T) {
		store := newMockStore()
		cacher := NewCacher(store)

		// 模拟缓存命中
		store.data["key1"] = "cached_value"
		store.On("Get", ctx, "key1", mock.AnythingOfType("*string")).Return(true, nil)

		var dst string
		found, err := cacher.Get(ctx, "key1", &dst, nil, nil)
		
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "cached_value", dst)
		
		store.AssertExpectations(t)
	})

	t.Run("cache miss with fallback", func(t *testing.T) {
		store := newMockStore()
		cacher := NewCacher(store)

		// 模拟缓存未命中
		store.On("Get", ctx, "key1", mock.AnythingOfType("*string")).Return(false, nil)
		
		// 模拟回退函数返回数据
		fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
			return "fallback_value", true, nil
		}
		
		// 模拟缓存存储回退结果
		store.On("MSet", ctx, map[string]interface{}{"key1": "fallback_value"}, time.Duration(0)).Return(nil)

		var dst string
		found, err := cacher.Get(ctx, "key1", &dst, fallback, nil)
		
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "fallback_value", dst)
		
		store.AssertExpectations(t)
	})

	t.Run("cache miss with fallback error", func(t *testing.T) {
		store := newMockStore()
		cacher := NewCacher(store)

		// 模拟缓存未命中
		store.On("Get", ctx, "key1", mock.AnythingOfType("*string")).Return(false, nil)
		
		// 模拟回退函数返回错误
		fallbackErr := errors.New("fallback error")
		fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
			return nil, false, fallbackErr
		}

		var dst string
		found, err := cacher.Get(ctx, "key1", &dst, fallback, nil)
		
		assert.Error(t, err)
		assert.False(t, found)
		assert.Equal(t, fallbackErr, err)
		
		store.AssertExpectations(t)
	})

	t.Run("cache error", func(t *testing.T) {
		store := newMockStore()
		cacher := NewCacher(store)

		// 模拟缓存错误
		cacheErr := errors.New("cache error")
		store.On("Get", ctx, "key1", mock.AnythingOfType("*string")).Return(false, cacheErr)

		var dst string
		found, err := cacher.Get(ctx, "key1", &dst, nil, nil)
		
		assert.Error(t, err)
		assert.False(t, found)
		assert.Equal(t, cacheErr, err)
		
		store.AssertExpectations(t)
	})
}

func TestCacher_MGet(t *testing.T) {
	ctx := context.Background()

	t.Run("all keys cached", func(t *testing.T) {
		mockStore := newMockStore()
		cacher := NewCacher(mockStore)

		// 模拟批量获取
		mockStore.data["key1"] = "value1"
		mockStore.data["key2"] = "value2"
		mockStore.On("MGet", ctx, []string{"key1", "key2"}, mock.Anything).Run(func(args mock.Arguments) {
			// 获取传递的dstMap参数
			dstMap := args.Get(2)
			
			// 使用反射填充dstMap
			dstMapValue := reflect.ValueOf(dstMap).Elem()
			resultMap := map[string]interface{}{"key1": "value1", "key2": "value2"}
			
			for k, v := range resultMap {
				keyValue := reflect.ValueOf(k)
				valueValue := reflect.ValueOf(v)
				dstMapValue.SetMapIndex(keyValue, valueValue)
			}
		}).Return(nil)

		result := make(map[string]string)
		err := cacher.MGet(ctx, []string{"key1", "key2"}, &result, nil, nil)
		
		assert.NoError(t, err)
		assert.Equal(t, "value1", result["key1"])
		assert.Equal(t, "value2", result["key2"])
		
		mockStore.AssertExpectations(t)
	})
}

func TestCacher_MDelete(t *testing.T) {
	ctx := context.Background()

	mockStore := newMockStore()
	cacher := NewCacher(mockStore)

	// 预先设置数据
	mockStore.data["key1"] = "value1"
	mockStore.data["key2"] = "value2"
	
	// 模拟删除
	mockStore.On("Del", ctx, []string{"key1", "key2"}).Return(int64(2), nil)

	count, err := cacher.MDelete(ctx, []string{"key1", "key2"})
	
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)
	
	mockStore.AssertExpectations(t)
}

func TestCacher_MRefresh(t *testing.T) {
	ctx := context.Background()

	mockStore := newMockStore()
	cacher := NewCacher(mockStore)

	// 模拟回退函数
	fallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		return map[string]interface{}{
			"key1": "new_value1",
			"key2": "new_value2",
		}, nil
	}

	// 模拟刷新存储
	mockStore.On("MSet", ctx, map[string]interface{}{"key1": "new_value1", "key2": "new_value2"}, time.Duration(0)).Return(nil)

	result := make(map[string]string)
	err := cacher.MRefresh(ctx, []string{"key1", "key2"}, &result, fallback, nil)
	
	assert.NoError(t, err)
	
	mockStore.AssertExpectations(t)
}