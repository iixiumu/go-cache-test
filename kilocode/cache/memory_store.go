package cache

import (
	"context"
	"encoding/json"
	"reflect"
	"sync"
	"time"
)

// memoryStore 实现了Store接口，使用内存作为存储后端
type memoryStore struct {
	data map[string][]byte
	mu   sync.RWMutex
}

// NewMemoryStore 创建一个新的内存Store实例
func NewMemoryStore() Store {
	return &memoryStore{
		data: make(map[string][]byte),
	}
}

// Get 从内存获取单个值
func (m *memoryStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	val, exists := m.data[key]
	if !exists {
		return false, nil
	}

	// 反序列化值到dst
	err := json.Unmarshal(val, dst)
	if err != nil {
		return false, err
	}

	return true, nil
}

// MGet 批量从内存获取值
func (m *memoryStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 确保dstMap是指向map的指针
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr || dstMapValue.Elem().Kind() != reflect.Map {
		return nil
	}

	// 获取map的元素类型
	dstMapElem := dstMapValue.Elem()
	mapValueType := dstMapElem.Type().Elem()

	// 批量获取值
	for _, key := range keys {
		val, exists := m.data[key]
		if !exists {
			continue
		}

		// 创建对应类型的值
		valuePtr := reflect.New(mapValueType).Interface()

		// 反序列化
		err := json.Unmarshal(val, valuePtr)
		if err != nil {
			continue
		}

		// 设置map值
		dstMapElem.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(valuePtr).Elem())
	}

	return nil
}

// Exists 批量检查键在内存中的存在性
func (m *memoryStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]bool)
	for _, key := range keys {
		_, exists := m.data[key]
		result[key] = exists
	}

	return result, nil
}

// MSet 批量设置键值对到内存
func (m *memoryStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for key, value := range items {
		// 序列化值
		valBytes, err := json.Marshal(value)
		if err != nil {
			return err
		}

		m.data[key] = valBytes

		// 如果设置了TTL，启动一个goroutine在TTL后删除键
		if ttl > 0 {
			go func(k string, duration time.Duration) {
				time.Sleep(duration)
				m.mu.Lock()
				delete(m.data, k)
				m.mu.Unlock()
			}(key, ttl)
		}
	}

	return nil
}

// Del 从内存删除指定键
func (m *memoryStore) Del(ctx context.Context, keys ...string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var deleted int64
	for _, key := range keys {
		if _, exists := m.data[key]; exists {
			delete(m.data, key)
			deleted++
		}
	}

	return deleted, nil
}
