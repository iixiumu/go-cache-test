package memory

import (
	"context"
	"reflect"
	"sync"
	"time"
	"github.com/xiumu/go-cache/store"
)

// MemoryStore 内存存储实现
type MemoryStore struct {
	data  map[string]interface{}
	mutex sync.RWMutex
}

// NewMemoryStore 创建新的内存存储实例
func NewMemoryStore() store.Store {
	return &MemoryStore{
		data: make(map[string]interface{}),
	}
}

// Get 从内存存储中获取单个值
func (m *MemoryStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	value, exists := m.data[key]
	if !exists {
		return false, nil
	}

	// 使用反射将value赋值给dst
	if dst != nil {
		dstValue := reflect.ValueOf(dst)
		if dstValue.Kind() != reflect.Ptr {
			return true, nil // dst不是指针，无法赋值
		}

		dstElem := dstValue.Elem()
		if !dstElem.CanSet() {
			return true, nil // dst元素不可设置
		}

		valueReflect := reflect.ValueOf(value)
		if !valueReflect.Type().AssignableTo(dstElem.Type()) {
			return true, nil // 类型不匹配，无法赋值
		}

		dstElem.Set(valueReflect)
	}

	return true, nil
}

// MGet 批量获取值到map中
func (m *MemoryStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// 检查dstMap是否为指针
	if dstMap == nil {
		return nil
	}

	mapValue := reflect.ValueOf(dstMap)
	if mapValue.Kind() != reflect.Ptr {
		return nil // dstMap不是指针
	}

	mapElem := mapValue.Elem()
	if mapElem.Kind() != reflect.Map {
		return nil // dstMap指向的不是map
	}

	// 确保map已初始化
	if mapElem.IsNil() {
		mapType := mapElem.Type()
		newMap := reflect.MakeMap(mapType)
		mapElem.Set(newMap)
	}

	// 填充map
	for _, key := range keys {
		if value, exists := m.data[key]; exists {
			keyValue := reflect.ValueOf(key)
			valueValue := reflect.ValueOf(value)

			// 检查类型兼容性
			mapKeyType := mapElem.Type().Key()
			mapValueType := mapElem.Type().Elem()

			if keyValue.Type().AssignableTo(mapKeyType) && valueValue.Type().AssignableTo(mapValueType) {
				mapElem.SetMapIndex(keyValue, valueValue)
			}
		}
	}

	return nil
}

// Exists 批量检查键存在性
func (m *MemoryStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make(map[string]bool)
	for _, key := range keys {
		_, exists := m.data[key]
		result[key] = exists
	}

	return result, nil
}

// MSet 批量设置键值对，支持TTL
func (m *MemoryStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for key, value := range items {
		m.data[key] = value
		// 在实际实现中，需要处理TTL
	}

	return nil
}

// Del 删除指定键
func (m *MemoryStore) Del(ctx context.Context, keys ...string) (int64, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var deleted int64
	for _, key := range keys {
		if _, exists := m.data[key]; exists {
			delete(m.data, key)
			deleted++
		}
	}

	return deleted, nil
}