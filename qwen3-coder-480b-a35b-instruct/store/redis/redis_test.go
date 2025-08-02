package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

func TestRedisStore(t *testing.T) {
	// 启动一个miniredis服务器用于测试
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer s.Close()

	// 创建Redis客户端连接到miniredis
	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	// 创建RedisStore实例
	store := New(client)

	// 测试MSet和Get
	ctx := context.Background()
	items := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	}

	// 测试MSet
	err = store.MSet(ctx, items, time.Minute)
	if err != nil {
		t.Errorf("MSet failed: %v", err)
	}

	// 测试Get
	var value string
	found, err := store.Get(ctx, "key1", &value)
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if !found {
		t.Error("Key should be found")
	}
	if value != "value1" {
		t.Errorf("Expected 'value1', got '%s'", value)
	}

	// 测试MGet
	resultMap := make(map[string]interface{})
	err = store.MGet(ctx, []string{"key1", "key2"}, &resultMap)
	if err != nil {
		t.Errorf("MGet failed: %v", err)
	}

	// 测试Exists
	exists, err := store.Exists(ctx, []string{"key1", "key2", "key3"})
	if err != nil {
		t.Errorf("Exists failed: %v", err)
	}
	if !exists["key1"] || !exists["key2"] || exists["key3"] {
		t.Error("Exists returned incorrect results")
	}

	// 测试Del
	count, err := store.Del(ctx, "key1", "key2")
	if err != nil {
		t.Errorf("Del failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected to delete 2 keys, got %d", count)
	}
}