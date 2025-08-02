package store

import (
	"context"
	"encoding/json"
	"reflect"
	"sync"
	"time"

	"github.com/xiumu/go-cache/cache"
)

// memoryStore 内存存储实现
type memoryStore struct {
	data map[string][]byte
	ttl  map[string]time.Time
	mu   sync.RWMutex
}

// NewMemoryStore 创建一个新的内存存储实例
func NewMemoryStore() cache.Store {
	return &memoryStore{
		data: make(map[string][]byte),
		ttl:  make(map[string]time.Time),
	}
}

// Get 从存储后端获取单个值
func (m *memoryStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 检查键是否存在
	data, exists := m.data[key]
	if !exists {
		return false, nil
	}

	// 检查是否过期
	if expire, ok := m.ttl[key]; ok && time.Now().After(expire) {
		// 过期了，删除键
		go m.deleteKey(key)
		return false, nil
	}

	// 反序列化数据到dst
	return true, json.Unmarshal(data, dst)
}

// MGet 批量获取值到map中
func (m *memoryStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 检查dstMap是否为指针
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr {
		return nil
	}

	// 获取map的实际值
	dstMapValue = dstMapValue.Elem()
	if dstMapValue.Kind() != reflect.Map {
		return nil
	}

	// 获取map的元素类型
	mapElemType := dstMapValue.Type().Elem()

	// 遍历所有键
	for _, key := range keys {
		// 检查键是否存在
		data, exists := m.data[key]
		if !exists {
			continue
		}

		// 检查是否过期
		if expire, ok := m.ttl[key]; ok && time.Now().After(expire) {
			// 过期了，删除键
			go m.deleteKey(key)
			continue
		}

		// 创建一个新的元素实例
		elem := reflect.New(mapElemType).Interface()

		// 反序列化数据
		err := json.Unmarshal(data, elem)
		if err != nil {
			continue
		}

		// 将值设置到map中
		dstMapValue.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(elem).Elem())
	}

	return nil
}

// Exists 批量检查键存在性
func (m *memoryStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]bool)
	now := time.Now()

	for _, key := range keys {
		// 检查键是否存在
		_, exists := m.data[key]
		if !exists {
			result[key] = false
			continue
		}

		// 检查是否过期
		if expire, ok := m.ttl[key]; ok && now.After(expire) {
			// 过期了
			result[key] = false
			// 删除键
			go m.deleteKey(key)
		} else {
			result[key] = true
		}
	}

	return result, nil
}

// MSet 批量设置键值对，支持TTL
func (m *memoryStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	expireTime := time.Time{}
	if ttl > 0 {
		expireTime = time.Now().Add(ttl)
	}

	for key, value := range items {
		// 序列化值
		data, err := json.Marshal(value)
		if err != nil {
			continue
		}

		// 存储数据
		m.data[key] = data

		// 设置过期时间
		if ttl > 0 {
			m.ttl[key] = expireTime
		} else {
			delete(m.ttl, key)
		}
	}

	return nil
}

// Del 删除指定键
func (m *memoryStore) Del(ctx context.Context, keys ...string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var deleted int64
	for _, key := range keys {
		if m.deleteKey(key) {
			deleted++
		}
	}

	return deleted, nil
}

// deleteKey 删除指定键（内部方法，需要在持有锁的情况下调用）
func (m *memoryStore) deleteKey(key string) bool {
	// 检查键是否存在
	_, exists := m.data[key]
	if !exists {
		return false
	}

	// 删除数据和过期时间
	delete(m.data, key)
	delete(m.ttl, key)

	return true
}