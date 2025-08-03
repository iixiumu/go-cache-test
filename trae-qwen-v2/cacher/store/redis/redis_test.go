package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"go-cache/cacher/store"
)

func TestRedisStore(t *testing.T) {
	// 创建miniredis实例
	mredis, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mredis.Close()

	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr: mredis.Addr(),
	})
	defer client.Close()

	// 创建RedisStore实例
	redisStore := NewRedisStore(client)

	// 创建StoreTester实例
	tester := store.NewStoreTester(redisStore)

	// 运行测试
	t.Run("Get", tester.TestGet)
	t.Run("MGet", tester.TestMGet)
	t.Run("Exists", tester.TestExists)
	t.Run("MSet", tester.TestMSet)
	t.Run("Del", tester.TestDel)
}

// 测试TTL功能
func TestRedisStoreTTL(t *testing.T) {
	// 创建miniredis实例
	mredis, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mredis.Close()

	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr: mredis.Addr(),
	})
	defer client.Close()

	// 创建RedisStore实例
	redisStore := NewRedisStore(client)

	ctx := context.Background()
	items := map[string]interface{}{
		"key1": "value1",
	}
	ttl := 1 * time.Second

	// 设置值并设置TTL
	if err := redisStore.MSet(ctx, items, ttl); err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 等待TTL过期
	time.Sleep(1*time.Second + 100*time.Millisecond)

	// 验证值已过期
	var result string
	found, err := redisStore.Get(ctx, "key1", &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if found {
		t.Fatalf("Get: expected key1 to be expired")
	}
}