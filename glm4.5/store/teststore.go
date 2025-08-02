package store

import (
	"context"
	"time"

	"go-cache/internal"
)

// TestStore 内存测试用的存储实现
type TestStore struct {
	data map[string][]byte
	ttl  map[string]time.Time
}

// NewTestStore 创建测试存储实例
func NewTestStore() *TestStore {
	return &TestStore{
		data: make(map[string][]byte),
		ttl:  make(map[string]time.Time),
	}
}

// Get 从存储后端获取单个值
func (s *TestStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	data, exists := s.data[key]
	if !exists {
		return false, nil
	}

	if ttl, hasTTL := s.ttl[key]; hasTTL && time.Now().After(ttl) {
		delete(s.data, key)
		delete(s.ttl, key)
		return false, nil
	}

	return true, internal.DeserializeValue(data, dst)
}

// MGet 批量获取值到map中
func (s *TestStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	_, _, err := internal.GetTypeOfMap(dstMap)
	if err != nil {
		return err
	}

	results := make(map[string]interface{})
	for _, key := range keys {
		data, exists := s.data[key]
		if !exists {
			continue
		}

		if ttl, hasTTL := s.ttl[key]; hasTTL && time.Now().After(ttl) {
			delete(s.data, key)
			delete(s.ttl, key)
			continue
		}

		var value interface{}
		if err := internal.DeserializeValue(data, &value); err != nil {
			continue
		}

		results[key] = value
	}

	for k, v := range results {
		if err := internal.SetMapValue(dstMap, k, v); err != nil {
			return err
		}
	}

	return nil
}

// Exists 批量检查键存在性
func (s *TestStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool)
	now := time.Now()

	for _, key := range keys {
		if _, exists := s.data[key]; !exists {
			result[key] = false
			continue
		}

		if ttl, hasTTL := s.ttl[key]; hasTTL && now.After(ttl) {
			delete(s.data, key)
			delete(s.ttl, key)
			result[key] = false
			continue
		}

		result[key] = true
	}

	return result, nil
}

// MSet 批量设置键值对，支持TTL
func (s *TestStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	for key, value := range items {
		data, err := internal.SerializeValue(value)
		if err != nil {
			return err
		}

		s.data[key] = data

		if ttl > 0 {
			s.ttl[key] = time.Now().Add(ttl)
		} else {
			delete(s.ttl, key)
		}
	}

	return nil
}

// Del 删除指定键
func (s *TestStore) Del(ctx context.Context, keys ...string) (int64, error) {
	var deleted int64
	for _, key := range keys {
		if _, exists := s.data[key]; exists {
			delete(s.data, key)
			delete(s.ttl, key)
			deleted++
		}
	}
	return deleted, nil
}