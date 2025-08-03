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
	// 启动miniredis服务器
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	// 创建Redis Store
	redisStore := NewRedisStore(client)

	// 创建测试套件并运行所有测试
	testSuite := store.NewStoreTestSuite(redisStore)
	
	// 运行所有测试
	t.Run("TestBasicGetSet", testSuite.TestBasicGetSet)
	t.Run("TestMGetMSet", testSuite.TestMGetMSet)
	t.Run("TestExists", testSuite.TestExists)
	t.Run("TestDel", testSuite.TestDel)
	t.Run("TestTTL", func(t *testing.T) {
		// 为TTL测试使用miniredis的FastForward功能
		ctx := context.Background()

		// 设置带TTL的键
		err := redisStore.MSet(ctx, map[string]interface{}{
			"ttl:test": "value",
		}, 100*time.Millisecond)
		if err != nil {
			t.Fatalf("MSet failed: %v", err)
		}

		// 立即获取应该存在
		var result string
		found, err := redisStore.Get(ctx, "ttl:test", &result)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if !found {
			t.Fatal("Expected to find key immediately")
		}

		// 使用miniredis的FastForward功能快速推进时间
		mr.FastForward(150 * time.Millisecond)

		// 再次获取应该不存在
		found, err = redisStore.Get(ctx, "ttl:test", &result)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if found {
			t.Fatal("Expected key to be expired")
		}
	})
	t.Run("TestComplexTypes", testSuite.TestComplexTypes)
	t.Run("TestNotFound", testSuite.TestNotFound)
}
