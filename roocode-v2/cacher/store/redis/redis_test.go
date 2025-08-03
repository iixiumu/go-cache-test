package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// TestRedisStore 测试RedisStore的实现
func TestRedisStore(t *testing.T) {
	// 创建一个miniredis实例用于测试
	mr := miniredis.RunT(t)

	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// 创建RedisStore
	store := NewRedisStore(client)

	// 创建StoreTester并运行所有测试
	tester := &store.StoreTester{
		NewStore: func() store.Store {
			return store
		},
	}

	tester.RunAllTests(t)
}

// TestRedisStoreSpecific 测试RedisStore的特定功能
func TestRedisStoreSpecific(t *testing.T) {
	// 创建一个miniredis实例用于测试
	mr := miniredis.RunT(t)

	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// 创建RedisStore
	store := NewRedisStore(client)
	ctx := context.Background()

	// 测试不同数据类型的存储和获取
	t.Run("TestDifferentTypes", func(t *testing.T) {
		// 存储字符串
		items := map[string]interface{}{
			"string_key": "string_value",
			"int_key":    42,
			"bool_key":   true,
			"float_key":  3.14,
		}

		err := store.MSet(ctx, items, 0)
		if err != nil {
			t.Fatalf("MSet failed: %v", err)
		}

		// 获取字符串
		var stringValue string
		found, err := store.Get(ctx, "string_key", &stringValue)
		if err != nil || !found || stringValue != "string_value" {
			t.Fatalf("Failed to get string_key")
		}

		// 获取整数
		var intValue int
		found, err = store.Get(ctx, "int_key", &intValue)
		if err != nil || !found || intValue != 42 {
			t.Fatalf("Failed to get int_key")
		}

		// 获取布尔值
		var boolValue bool
		found, err = store.Get(ctx, "bool_key", &boolValue)
		if err != nil || !found || !boolValue {
			t.Fatalf("Failed to get bool_key")
		}

		// 获取浮点数
		var floatValue float64
		found, err = store.Get(ctx, "float_key", &floatValue)
		if err != nil || !found || floatValue != 3.14 {
			t.Fatalf("Failed to get float_key")
		}
	})

	// 测试TTL功能
	t.Run("TestTTL", func(t *testing.T) {
		// 设置带TTL的值
		items := map[string]interface{}{
			"ttl_key": "ttl_value",
		}

		err := store.MSet(ctx, items, time.Second)
		if err != nil {
			t.Fatalf("MSet failed: %v", err)
		}

		// 验证值存在
		var value string
		found, err := store.Get(ctx, "ttl_key", &value)
		if err != nil || !found || value != "ttl_value" {
			t.Fatalf("Failed to get ttl_key")
		}

		// 快进时间
		mr.FastForward(time.Second * 2)

		// 验证值已过期
		found, err = store.Get(ctx, "ttl_key", &value)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if found {
			t.Fatalf("Expected ttl_key to be expired")
		}
	})
}
