package cache

import (
	"context"
	"encoding/json"
	"reflect"
	"sync"
	"time"
)

// memoryStore 实现了Store接口的内存存储
type memoryStore struct {
	data map[string]*memoryItem
	mu   sync.RWMutex
}

// memoryItem 内存中的存储项
type memoryItem struct {
	Data    []byte
	Expires time.Time
}

// NewMemoryStore 创建一个新的内存存储实例
func NewMemoryStore() Store {
	return &memoryStore{
		data: make(map[string]*memoryItem),
	}
}

// Get 从内存存储中获取单个值
func (m *memoryStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	m.mu.RLock()
	item, exists := m.data[key]
	m.mu.RUnlock()

	if !exists {
		return false, nil
	}

	// 检查是否过期
	if !item.Expires.IsZero() && time.Now().After(item.Expires) {
		m.mu.Lock()
		delete(m.data, key)
		m.mu.Unlock()
		return false, nil
	}

	// 反序列化数据到dst
	return true, json.Unmarshal(item.Data, dst)
}

// MGet 批量从内存存储中获取值
func (m *memoryStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr || dstMapValue.IsNil() {
		return nil
	}

	dstMapElem := dstMapValue.Elem()
	if dstMapElem.Kind() != reflect.Map {
		return nil
	}

	mapKeyType := dstMapElem.Type().Key()
	mapValueType := dstMapElem.Type().Elem()

	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()
	for _, key := range keys {
		item, exists := m.data[key]
		if !exists {
			continue
		}

		// 检查是否过期
		if !item.Expires.IsZero() && now.After(item.Expires) {
			continue
		}

		// 创建目标值
		elemValue := reflect.New(mapValueType).Interface()

		// 反序列化数据
		err := json.Unmarshal(item.Data, elemValue)
		if err != nil {
			continue
		}

		// 设置到map中
		dstMapElem.SetMapIndex(reflect.ValueOf(key).Convert(mapKeyType), reflect.ValueOf(elemValue).Elem())
	}

	return nil
}

// Exists 批量检查键存在性
func (m *memoryStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool)
	now := time.Now()

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, key := range keys {
		item, exists := m.data[key]
		if !exists {
			result[key] = false
			continue
		}

		// 检查是否过期
		if !item.Expires.IsZero() && now.After(item.Expires) {
			result[key] = false
			continue
		}

		result[key] = true
	}

	return result, nil
}

// MSet 批量设置键值对
func (m *memoryStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var expires time.Time
	if ttl > 0 {
		expires = time.Now().Add(ttl)
	}

	for key, value := range items {
		data, err := json.Marshal(value)
		if err != nil {
			continue
		}

		m.data[key] = &memoryItem{
			Data:    data,
			Expires: expires,
		}
	}

	return nil
}

// Del 删除指定键
func (m *memoryStore) Del(ctx context.Context, keys ...string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var count int64
	for _, key := range keys {
		if _, exists := m.data[key]; exists {
			delete(m.data, key)
			count++
		}
	}

	return count, nil
}