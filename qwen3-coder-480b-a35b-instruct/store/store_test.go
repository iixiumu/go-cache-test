package store

import (
	"context"
	"testing"
	"time"
)

func TestStoreInterface(t *testing.T) {
	// 这是一个接口测试，用于确保所有实现都符合Store接口
	// 实际的实现测试应该在各自的实现包中进行
	var _ Store = &mockStore{}
}

type mockStore struct{}

func (m *mockStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	return false, nil
}

func (m *mockStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	return nil
}

func (m *mockStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	return nil, nil
}

func (m *mockStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	return nil
}

func (m *mockStore) Del(ctx context.Context, keys ...string) (int64, error) {
	return 0, nil
}