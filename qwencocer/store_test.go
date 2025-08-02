package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// mockStore is a mock implementation of the Store interface for testing
type mockStore struct {
	getFunc    func(ctx context.Context, key string, dst interface{}) (bool, error)
	mgetFunc   func(ctx context.Context, keys []string, dstMap interface{}) error
	existsFunc func(ctx context.Context, keys []string) (map[string]bool, error)
	msetFunc   func(ctx context.Context, items map[string]interface{}, ttl time.Duration) error
	delFunc    func(ctx context.Context, keys ...string) (int64, error)
}

func (m *mockStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, key, dst)
	}
	return false, nil
}

func (m *mockStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	if m.mgetFunc != nil {
		return m.mgetFunc(ctx, keys, dstMap)
	}
	return nil
}

func (m *mockStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	if m.existsFunc != nil {
		return m.existsFunc(ctx, keys)
	}
	return map[string]bool{}, nil
}

func (m *mockStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if m.msetFunc != nil {
		return m.msetFunc(ctx, items, ttl)
	}
	return nil
}

func (m *mockStore) Del(ctx context.Context, keys ...string) (int64, error) {
	if m.delFunc != nil {
		return m.delFunc(ctx, keys...)
	}
	return 0, nil
}

func TestStoreInterface(t *testing.T) {
	// This test ensures that our mockStore implements the Store interface
	var _ Store = &mockStore{}
	assert.True(t, true) // This is just to satisfy the linter
}