package cacher

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCacher_Get(t *testing.T) {
	// 创建一个简单的内存Store用于测试
	store := &simpleStore{}
	cacher := NewCacher(store)

	ctx := context.Background()

	// 测试缓存命中
	t.Run("cache hit", func(t *testing.T) {
		// 先设置缓存
		err := store.MSet(ctx, map[string]interface{}{
			"key1": "value1",
		}, time.Hour)
		assert.NoError(t, err)

		var result string
		found, err := cacher.Get(ctx, "key1", &result, nil, nil)
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "value1", result)
	})

	// 测试缓存未命中，回退函数返回值
	t.Run("cache miss with fallback", func(t *testing.T) {
		var result string
		fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
			return "fallback_value", true, nil
		}
		found, err := cacher.Get(ctx, "key2", &result, fallback, nil)
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "fallback_value", result)
	})
}

func TestCacher_MGet(t *testing.T) {
	// 创建一个简单的内存Store用于测试
	store := &simpleStore{}
	cacher := NewCacher(store)

	ctx := context.Background()

	// 测试部分命中
	t.Run("partial cache hit", func(t *testing.T) {
		// 先设置部分缓存
		err := store.MSet(ctx, map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		}, time.Hour)
		assert.NoError(t, err)

		resultMap := make(map[string]string)
		fallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
			return map[string]interface{}{
				"key3": "fallback_value3",
			}, nil
		}
		err = cacher.MGet(ctx, []string{"key1", "key2", "key3"}, &resultMap, fallback, nil)
		assert.NoError(t, err)
		assert.Len(t, resultMap, 3)
		assert.Equal(t, "value1", resultMap["key1"])
		assert.Equal(t, "value2", resultMap["key2"])
		assert.Equal(t, "fallback_value3", resultMap["key3"])
	})
}

// simpleStore 是一个用于测试的简单Store实现
type simpleStore struct {
	data map[string]interface{}
}

func (s *simpleStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	if s.data == nil {
		s.data = make(map[string]interface{})
	}

	value, exists := s.data[key]
	if !exists {
		return false, nil
	}

	// 直接设置值（简化处理）
	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() == reflect.Ptr && !dstValue.IsNil() {
		dstElem := dstValue.Elem()
		valueReflect := reflect.ValueOf(value)
		if valueReflect.Type().AssignableTo(dstElem.Type()) {
			dstElem.Set(valueReflect)
		}
	}

	return true, nil
}

func (s *simpleStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	if s.data == nil {
		s.data = make(map[string]interface{})
	}

	// 将结果填充到dstMap中
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr || dstMapValue.IsNil() {
		return nil
	}

	mapValue := dstMapValue.Elem()

	// 假设dstMap是一个map[string]interface{}类型
	if mapValue.Kind() == reflect.Map && mapValue.Type().Key().Kind() == reflect.String {
		for _, key := range keys {
			if value, exists := s.data[key]; exists {
				mapValue.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
			}
		}
	}

	return nil
}

func (s *simpleStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	if s.data == nil {
		s.data = make(map[string]interface{})
	}

	result := make(map[string]bool)
	for _, key := range keys {
		_, exists := s.data[key]
		result[key] = exists
	}
	return result, nil
}

func (s *simpleStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if s.data == nil {
		s.data = make(map[string]interface{})
	}

	for key, value := range items {
		s.data[key] = value
	}
	return nil
}

func (s *simpleStore) Del(ctx context.Context, keys ...string) (int64, error) {
	if s.data == nil {
		s.data = make(map[string]interface{})
	}

	count := int64(0)
	for _, key := range keys {
		if _, exists := s.data[key]; exists {
			delete(s.data, key)
			count++
		}
	}
	return count, nil
}
