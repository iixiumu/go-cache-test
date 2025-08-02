package cache

import (
	"context"
	"encoding/json"
	"reflect"
	"sync"
	"time"
)

// testMemoryStore 用于测试的内存存储实现
type testMemoryStore struct {
	data map[string][]byte
	ttl  map[string]time.Time
	mu   sync.RWMutex
}

// newTestMemoryStore 创建一个新的测试内存存储实例
func newTestMemoryStore() *testMemoryStore {
	return &testMemoryStore{
		data: make(map[string][]byte),
		ttl:  make(map[string]time.Time),
	}
}

// Get 从存储后端获取单个值
func (m *testMemoryStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 检查键是否存在
	data, exists := m.data[key]
	if !exists {
		return false, nil
	}

	// 检查是否过期
	if expire, ok := m.ttl[key]; ok && time.Now().After(expire) {
		return false, nil
	}

	// 反序列化数据到dst
	return true, json.Unmarshal(data, dst)
}

// MGet 批量获取值到map中
func (m *testMemoryStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 简化实现，只处理字符串map
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr {
		return nil
	}

	dstMapValue = dstMapValue.Elem()
	if dstMapValue.Kind() != reflect.Map {
		return nil
	}

	for _, key := range keys {
		data, exists := m.data[key]
		if !exists {
			continue
		}

		// 检查是否过期
		if expire, ok := m.ttl[key]; ok && time.Now().After(expire) {
			continue
		}

		var value string
		err := json.Unmarshal(data, &value)
		if err != nil {
			continue
		}

		dstMapValue.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
	}

	return nil
}

// Exists 批量检查键存在性
func (m *testMemoryStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]bool)
	now := time.Now()

	for _, key := range keys {
		_, exists := m.data[key]
		if !exists {
			result[key] = false
			continue
		}

		// 检查是否过期
		if expire, ok := m.ttl[key]; ok && now.After(expire) {
			result[key] = false
		} else {
			result[key] = true
		}
	}

	return result, nil
}

// MSet 批量设置键值对，支持TTL
func (m *testMemoryStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	expireTime := time.Time{}
	if ttl > 0 {
		expireTime = time.Now().Add(ttl)
	}

	for key, value := range items {
		data, err := json.Marshal(value)
		if err != nil {
			continue
		}

		m.data[key] = data

		if ttl > 0 {
			m.ttl[key] = expireTime
		} else {
			delete(m.ttl, key)
		}
	}

	return nil
}

// Del 删除指定键
func (m *testMemoryStore) Del(ctx context.Context, keys ...string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var deleted int64
	for _, key := range keys {
		if _, exists := m.data[key]; exists {
			delete(m.data, key)
			delete(m.ttl, key)
			deleted++
		}
	}

	return deleted, nil
}