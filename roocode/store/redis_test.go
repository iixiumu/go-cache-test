package store

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

func TestRedisStore(t *testing.T) {
	// 创建一个miniredis实例用于测试
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to create miniredis: %v", err)
	}
	defer s.Close()

	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	// 创建Redis存储实例
	store := NewRedisStore(client)
	ctx := context.Background()

	// 测试MSet和Get
	items := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}

	err = store.MSet(ctx, items, time.Second*10)
	if err != nil {
		t.Fatalf("Failed to MSet: %v", err)
	}

	// 测试Get
	var value string
	found, err := store.Get(ctx, "key1", &value)
	if err != nil {
		t.Fatalf("Failed to Get: %v", err)
	}
	if !found {
		t.Fatalf("Key not found")
	}
	if value != "value1" {
		t.Fatalf("Expected 'value1', got '%s'", value)
	}

	// 测试Exists
	exists, err := store.Exists(ctx, []string{"key1", "key2", "nonexistent"})
	if err != nil {
		t.Fatalf("Failed to Exists: %v", err)
	}
	if !exists["key1"] || !exists["key2"] || exists["nonexistent"] {
		t.Fatalf("Exists returned unexpected results: %v", exists)
	}

	// 测试Del
	count, err := store.Del(ctx, "key1", "key2")
	if err != nil {
		t.Fatalf("Failed to Del: %v", err)
	}
	if count != 2 {
		t.Fatalf("Expected to delete 2 keys, got %d", count)
	}
}
