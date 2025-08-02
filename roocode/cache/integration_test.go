package cache

import (
	"context"
	"testing"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/xiumu/go-cache/store"
)

// User 用户结构体用于测试
type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestCacheIntegration(t *testing.T) {
	// 创建一个Ristretto缓存实例用于测试
	rcache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1000,
		MaxCost:     10000,
		BufferItems: 64,
	})
	if err != nil {
		t.Fatalf("Failed to create Ristretto cache: %v", err)
	}

	// 创建存储实例
	s := store.NewRistrettoStore(rcache)

	// 创建缓存实例
	c := New(s)
	ctx := context.Background()

	// 测试完整的缓存生命周期
	t.Run("完整缓存生命周期", func(t *testing.T) {
		// 1. 获取不存在的键，应该调用回退函数
		var user User
		calls := 0
		fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
			calls++
			if key == "user:1" {
				return User{ID: 1, Name: "Alice", Age: 25}, true, nil
			}
			return nil, false, nil
		}

		found, err := c.Get(ctx, "user:1", &user, fallback, nil)
		if err != nil {
			t.Fatalf("Failed to Get: %v", err)
		}
		if !found {
			t.Fatalf("Key not found")
		}
		if calls != 1 {
			t.Fatalf("Expected fallback to be called once, got %d", calls)
		}
		if user.Name != "Alice" {
			t.Fatalf("Expected 'Alice', got '%s'", user.Name)
		}

		// 等待缓存写入完成
		time.Sleep(time.Millisecond * 10)

		// 2. 再次获取，应该从缓存中获取，回退函数不应该被调用
		calls = 0
		found, err = c.Get(ctx, "user:1", &user, fallback, nil)
		if err != nil {
			t.Fatalf("Failed to Get: %v", err)
		}
		if !found {
			t.Fatalf("Key not found in cache")
		}
		if calls != 0 {
			t.Fatalf("Expected fallback not to be called, got %d", calls)
		}
		if user.Name != "Alice" {
			t.Fatalf("Expected 'Alice', got '%s'", user.Name)
		}

		// 3. 带TTL的缓存
		var userWithTTL User
		opts := &CacheOptions{TTL: time.Millisecond * 100}
		calls = 0
		ttlFallback := func(ctx context.Context, key string) (interface{}, bool, error) {
			calls++
			if key == "user:ttl" {
				return User{ID: 2, Name: "Bob", Age: 30}, true, nil
			}
			return nil, false, nil
		}

		found, err = c.Get(ctx, "user:ttl", &userWithTTL, ttlFallback, opts)
		if err != nil {
			t.Fatalf("Failed to Get with TTL: %v", err)
		}
		if !found {
			t.Fatalf("Key not found")
		}
		if calls != 1 {
			t.Fatalf("Expected TTL fallback to be called once, got %d", calls)
		}
		if userWithTTL.Name != "Bob" {
			t.Fatalf("Expected 'Bob', got '%s'", userWithTTL.Name)
		}

		// 等待TTL过期
		time.Sleep(time.Millisecond * 150)

		// 4. TTL过期后再次获取，应该再次调用回退函数
		calls = 0
		found, err = c.Get(ctx, "user:ttl", &userWithTTL, ttlFallback, opts)
		if err != nil {
			t.Fatalf("Failed to Get with expired TTL: %v", err)
		}
		if !found {
			t.Fatalf("Key not found after TTL expiration")
		}
		if calls != 1 {
			t.Fatalf("Expected TTL fallback to be called again, got %d", calls)
		}
		if userWithTTL.Name != "Bob" {
			t.Fatalf("Expected 'Bob', got '%s'", userWithTTL.Name)
		}

		// 5. 批量操作
		keys := []string{"user:1", "user:2", "user:3"}
		users := make(map[string]User)
		batchCalls := 0
		batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
			batchCalls++
			result := make(map[string]interface{})
			for i, key := range keys {
				if key == "user:2" || key == "user:3" {
					id := i + 1
					result[key] = User{ID: id, Name: "User" + key[5:], Age: 20 + id}
				}
			}
			return result, nil
		}

		err = c.MGet(ctx, keys, &users, batchFallback, nil)
		if err != nil {
			t.Fatalf("Failed to MGet: %v", err)
		}
		if batchCalls != 1 {
			t.Fatalf("Expected batch fallback to be called once, got %d", batchCalls)
		}
		if len(users) != 3 {
			t.Fatalf("Expected 3 users, got %d", len(users))
		}

		// 6. 批量删除
		count, err := c.MDelete(ctx, []string{"user:1", "user:2"})
		if err != nil {
			t.Fatalf("Failed to MDelete: %v", err)
		}
		if count != 2 {
			t.Fatalf("Expected to delete 2 keys, got %d", count)
		}

		// 7. 验证删除
		found, err = c.Get(ctx, "user:1", &user, nil, nil)
		if err != nil {
			t.Fatalf("Failed to Get: %v", err)
		}
		if found {
			t.Fatalf("Key should have been deleted")
		}

		// 8. 批量刷新
		users = make(map[string]User)
		refreshCalls := 0
		refreshFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
			refreshCalls++
			result := make(map[string]interface{})
			for i, key := range keys {
				id := i + 1
				result[key] = User{ID: id, Name: "RefreshedUser" + key[5:], Age: 25 + id}
			}
			return result, nil
		}

		err = c.MRefresh(ctx, []string{"user:1", "user:2"}, &users, refreshFallback, nil)
		if err != nil {
			t.Fatalf("Failed to MRefresh: %v", err)
		}
		if refreshCalls != 1 {
			t.Fatalf("Expected refresh fallback to be called once, got %d", refreshCalls)
		}
		if len(users) != 2 {
			t.Fatalf("Expected 2 users, got %d", len(users))
		}
	})
}
