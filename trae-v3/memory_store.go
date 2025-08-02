package deepseek

import (
	"context"
	"sync"
	"time"
)

// memoryStore 实现Store接口的内存存储
type memoryStore struct {
	items map[string]item
	mu    sync.RWMutex
}

type item struct {
	value interface{}
	exp   time.Time
}

// NewMemoryStore 创建新的内存存储实例
func NewMemoryStore() Store {
	return &memoryStore{
		items: make(map[string]item),
	}
}

func (m *memoryStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	it, ok := m.items[key]
	if !ok || (!it.exp.IsZero() && time.Now().After(it.exp)) {
		return false, nil
	}

	// 这里应该使用反射将值赋给dst
	return true, nil
}

func (m *memoryStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	// 实现批量获取逻辑
	return nil
}

func (m *memoryStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	// 实现存在性检查
	return nil, nil
}

func (m *memoryStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	// 实现批量设置
	return nil
}

func (m *memoryStore) Del(ctx context.Context, keys ...string) (int64, error) {
	// 实现删除逻辑
	return 0, nil
}