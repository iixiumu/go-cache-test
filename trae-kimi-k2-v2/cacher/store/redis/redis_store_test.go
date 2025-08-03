package redis

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	redisclient "github.com/redis/go-redis/v9"
)

func TestRedisStore(t *testing.T) {
	// 创建miniredis实例
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis run failed: %v", err)
	}
	defer mr.Close()

	// 创建Redis客户端
	client := redisclient.NewClient(&redisclient.Options{
		Addr: mr.Addr(),
	})
	
	// 创建RedisStore实例
	redisStore := NewRedisStore(client)
	defer redisStore.Close()
	
	// 运行统一的测试套件
	testSuite := store.NewStoreTestSuite(redisStore, t)
	testSuite.Run()
}

func TestRedisStoreWithRealRedis(t *testing.T) {
	// 跳过，需要真实的Redis服务器
	t.Skip("Skip real Redis test, requires Redis server")
}

func TestRedisStore_EmptyOperations(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis run failed: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	redisStore := NewRedisStore(client)
	defer redisStore.Close()
	ctx := context.Background()

	// 测试空操作
	err = redisStore.MSet(ctx, map[string]interface{}{}, 0)
	if err != nil {
		t.Errorf("Empty MSet should not fail: %v", err)
	}

	exists, err := redisStore.Exists(ctx, []string{})
	if err != nil {
		t.Errorf("Empty Exists should not fail: %v", err)
	}
	if len(exists) != 0 {
		t.Errorf("Empty Exists should return empty map")
	}

	deleted, err := redisStore.Del(ctx)
	if err != nil {
		t.Errorf("Empty Del should not fail: %v", err)
	}
	if deleted != 0 {
		t.Errorf("Empty Del should return 0")
	}
}

func TestRedisStore_ErrorCases(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis run failed: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	redisStore := NewRedisStore(client)
	defer redisStore.Close()
	ctx := context.Background()

	// 测试无效的JSON
	mr.Set("invalid_json", "{invalid json}")
	
	var val string
	found, err := redisStore.Get(ctx, "invalid_json", &val)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
	if found {
		t.Error("Should not be found for invalid JSON")
	}
}

func TestRedisStore_Close(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis run failed: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	redisStore := NewRedisStore(client)
	
	err = redisStore.Close()
	if err != nil {
		t.Errorf("Close should not fail: %v", err)
	}
}