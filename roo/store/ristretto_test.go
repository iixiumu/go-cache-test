package store

import (
	"context"
	"testing"
	"time"

	"github.com/dgraph-io/ristretto"
)

func TestRistrettoStore(t *testing.T) {
	// 创建一个Ristretto缓存实例用于测试
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1000,
		MaxCost:     10000,
		BufferItems: 64,
	})
	if err != nil {
		t.Fatalf("Failed to create Ristretto cache: %v", err)
	}

	// 创建Ristretto存储实例
	store := NewRistrettoStore(cache)
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

	// 等待一点时间让缓存写入完成
	time.Sleep(time.Millisecond * 10)

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
	// 注意：Ristretto的Del方法不返回实际删除的数量，所以我们不检查count的值
	_ = count

	// 等待删除操作完成
	time.Sleep(time.Millisecond * 10)

	// 验证键已被删除
	found, err = store.Get(ctx, "key1", &value)
	if err != nil {
		t.Fatalf("Failed to Get: %v", err)
	}
	if found {
		t.Fatalf("Key should have been deleted")
	}
}
