package cache

import (
	"context"
	"reflect"
	"time"
)

// MockStore 模拟Store实现，用于测试和示例
type MockStore struct {
	data map[string]interface{}
}

// NewMockStore 创建新的模拟存储实例
func NewMockStore() Store {
	return &MockStore{
		data: make(map[string]interface{}),
	}
}

// SetData 设置数据到模拟存储
func (m *MockStore) SetData(key string, value interface{}) {
	m.data[key] = value
}

func (m *MockStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	value, exists := m.data[key]
	if !exists {
		return false, nil
	}

	// 简单的类型转换，实际实现中应该使用JSON序列化
	if err := setValue(dst, value); err != nil {
		return false, err
	}

	return true, nil
}

func (m *MockStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	dstValue := reflect.ValueOf(dstMap)
	if dstValue.Kind() != reflect.Ptr || dstValue.Elem().Kind() != reflect.Map {
		return ErrInvalidDstMap
	}

	dstMapValue := dstValue.Elem()
	if dstMapValue.IsNil() {
		dstMapValue.Set(reflect.MakeMap(dstMapValue.Type()))
	}

	for _, key := range keys {
		if value, exists := m.data[key]; exists {
			dstMapValue.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
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
	deleted := int64(0)
	for _, key := range keys {
		if _, exists := m.data[key]; exists {
			delete(m.data, key)
			deleted++
		}
	}
	return deleted, nil
}
