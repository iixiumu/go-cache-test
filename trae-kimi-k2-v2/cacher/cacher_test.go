package cacher

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-cache/cacher/store/redis"
	"go-cache/cacher/store/ristretto"

	"github.com/alicebob/miniredis/v2"
	redisclient "github.com/redis/go-redis/v9"
)

// MockStore 用于测试的mock存储
type MockStore struct {
	data    map[string]interface{}
	exists  map[string]bool
	delKeys []string
}

func NewMockStore() *MockStore {
	return &MockStore{
		data:    make(map[string]interface{}),
		exists:  make(map[string]bool),
		delKeys: make([]string, 0),
	}
}

func (m *MockStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	if val, exists := m.data[key]; exists {
		// 简化处理，实际应该使用反射
		return true, nil
	}
	return false, nil
}

func (m *MockStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	return nil
}

func (m *MockStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	results := make(map[string]bool)
	for _, key := range keys {
		_, exists := m.data[key]
		results[key] = exists
	}
	return results, nil
}

func (m *MockStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	for key, value := range items {
		m.data[key] = value
		m.exists[key] = true
	}
	return nil
}

func (m *MockStore) Del(ctx context.Context, keys ...string) (int64, error) {
	deleted := int64(0)
	for _, key := range keys {
		if _, exists := m.data[key]; exists {
			delete(m.data, key)
			delete(m.exists, key)
			deleted++
		}
		m.delKeys = append(m.delKeys, key)
	}
	return deleted, nil
}

func (m *MockStore) Close() error {
	return nil
}

func TestCacherWithRedisStore(t *testing.T) {
	// 使用miniredis测试RedisStore
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis run failed: %v", err)
	}
	defer mr.Close()

	client := redisclient.NewClient(&redisclient.Options{
		Addr: mr.Addr(),
	})
	redisStore := redis.NewRedisStore(client)
	cacher := NewCacher(redisStore)

	runCacherTests(t, cacher)
}

func TestCacherWithRistrettoStore(t *testing.T) {
	ristrettoStore, err := ristretto.NewRistrettoStore()
	if err != nil {
		t.Fatalf("NewRistrettoStore failed: %v", err)
	}
	defer ristrettoStore.Close()

	cacher := NewCacher(ristrettoStore)
	runCacherTests(t, cacher)
}

func runCacherTests(t *testing.T, cacher Cacher) {
	ctx := context.Background()

	t.Run("Get with cache hit", func(t *testing.T) {
		// 预设置缓存
		fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
			return "fallback_value", true, nil
		}

		var result string
		found, err := cacher.Get(ctx, "test_key", &result, fallback, &CacheOptions{TTL: time.Minute})
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if !found {
			t.Error("Expected found")
		}
		if result != "fallback_value" {
			t.Errorf("Expected 'fallback_value', got '%s'", result)
		}

		// 再次获取，应该命中缓存
		var cachedResult string
		found, err = cacher.Get(ctx, "test_key", &cachedResult, func(ctx context.Context, key string) (interface{}, bool, error) {
			return "should_not_be_called", true, nil
		}, &CacheOptions{TTL: time.Minute})
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if !found {
			t.Error("Expected found from cache")
		}
		if cachedResult != "fallback_value" {
			t.Errorf("Expected cached value 'fallback_value', got '%s'", cachedResult)
		}
	})

	t.Run("Get with cache miss and fallback not found", func(t *testing.T) {
		fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
			return nil, false, nil
		}

		var result string
		found, err := cacher.Get(ctx, "nonexistent_key", &result, fallback, nil)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if found {
			t.Error("Expected not found")
		}
	})

	t.Run("Get with fallback error", func(t *testing.T) {
		fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
			return nil, false, errors.New("fallback error")
		}

		var result string
		_, err := cacher.Get(ctx, "error_key", &result, fallback, nil)
		if err == nil {
			t.Error("Expected error from fallback")
		}
	})

	t.Run("MGet with partial cache hit", func(t *testing.T) {
		// 预设置部分缓存
		batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
			result := make(map[string]interface{})
			for _, key := range keys {
				result[key] = "fallback_" + key
			}
			return result, nil
		}

		// 先设置一些缓存
		fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
			return "cached_value", true, nil
		}
		var dummy string
		cacher.Get(ctx, "cached_key", &dummy, fallback, &CacheOptions{TTL: time.Minute})

		// 批量获取
		keys := []string{"cached_key", "new_key1", "new_key2"}
		var results map[string]interface{}
		err := cacher.MGet(ctx, keys, &results, batchFallback, &CacheOptions{TTL: time.Minute})
		if err != nil {
			t.Fatalf("MGet failed: %v", err)
		}

		if len(results) != 3 {
			t.Errorf("Expected 3 results, got %d", len(results))
		}

		// 验证结果
		if results["cached_key"] != "cached_value" {
			t.Errorf("Expected cached value")
		}
		if results["new_key1"] != "fallback_new_key1" {
			t.Errorf("Expected fallback value")
		}
		if results["new_key2"] != "fallback_new_key2" {
			t.Errorf("Expected fallback value")
		}
	})

	t.Run("MGet with batch fallback error", func(t *testing.T) {
		batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
			return nil, errors.New("batch fallback error")
		}

		keys := []string{"key1", "key2"}
		var results map[string]interface{}
		err := cacher.MGet(ctx, keys, &results, batchFallback, nil)
		if err == nil {
			t.Error("Expected error from batch fallback")
		}
	})

	t.Run("MDelete", func(t *testing.T) {
		// 先设置缓存
		fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
			return "value", true, nil
		}
		var dummy string
		cacher.Get(ctx, "del_key1", &dummy, fallback, &CacheOptions{TTL: time.Minute})
		cacher.Get(ctx, "del_key2", &dummy, fallback, &CacheOptions{TTL: time.Minute})

		// 删除缓存
		deleted, err := cacher.MDelete(ctx, []string{"del_key1", "del_key2", "nonexistent"})
		if err != nil {
			t.Fatalf("MDelete failed: %v", err)
		}
		if deleted != 2 {
			t.Errorf("Expected 2 deleted, got %d", deleted)
		}

		// 验证删除
		var result string
		found, err := cacher.Get(ctx, "del_key1", &result, func(ctx context.Context, key string) (interface{}, bool, error) {
			return "new_value", true, nil
		}, nil)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if !found {
			t.Error("Expected found after fallback")
		}
		if result != "new_value" {
			t.Errorf("Expected 'new_value', got '%s'", result)
		}
	})

	t.Run("MRefresh", func(t *testing.T) {
		// 先设置缓存
		fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
			return "old_value", true, nil
		}
		var dummy string
		cacher.Get(ctx, "refresh_key", &dummy, fallback, &CacheOptions{TTL: time.Minute})

		// 强制刷新
		refreshFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
			result := make(map[string]interface{})
			for _, key := range keys {
				result[key] = "refreshed_value"
			}
			return result, nil
		}

		keys := []string{"refresh_key"}
		var results map[string]interface{}
		err := cacher.MRefresh(ctx, keys, &results, refreshFallback, &CacheOptions{TTL: time.Minute})
		if err != nil {
			t.Fatalf("MRefresh failed: %v", err)
		}

		if results["refresh_key"] != "refreshed_value" {
			t.Errorf("Expected refreshed value")
		}

		// 验证刷新后的值
		var refreshedResult string
		found, err := cacher.Get(ctx, "refresh_key", &refreshedResult, func(ctx context.Context, key string) (interface{}, bool, error) {
			return "should_not_be_called", true, nil
		}, &CacheOptions{TTL: time.Minute})
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if !found {
			t.Error("Expected found after refresh")
		}
		if refreshedResult != "refreshed_value" {
			t.Errorf("Expected 'refreshed_value', got '%s'", refreshedResult)
		}
	})

	t.Run("Empty operations", func(t *testing.T) {
		// 测试空键列表
		var results map[string]interface{}
		err := cacher.MGet(ctx, []string{}, &results, func(ctx context.Context, keys []string) (map[string]interface{}, error) {
			return nil, nil
		}, nil)
		if err != nil {
			t.Errorf("MGet with empty keys should not fail: %v", err)
		}

		deleted, err := cacher.MDelete(ctx, []string{})
		if err != nil {
			t.Errorf("MDelete with empty keys should not fail: %v", err)
		}
		if deleted != 0 {
			t.Errorf("Expected 0 deleted for empty keys")
		}

		err = cacher.MRefresh(ctx, []string{}, &results, func(ctx context.Context, keys []string) (map[string]interface{}, error) {
			return nil, nil
		}, nil)
		if err != nil {
			t.Errorf("MRefresh with empty keys should not fail: %v", err)
		}
	})
}