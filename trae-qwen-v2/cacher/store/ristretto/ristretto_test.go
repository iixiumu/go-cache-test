package ristretto

import (
	"context"
	"testing"
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"go-cache/cacher/store"
)

func TestRistrettoStore(t *testing.T) {
	// 创建Ristretto缓存实例
	cache, err := ristretto.NewCache(&ristretto.Config[string, interface{}]{
		NumCounters: 1000,
		MaxCost:     100,
		BufferItems: 64,
	})
	if err != nil {
		t.Fatalf("Failed to create Ristretto cache: %v", err)
	}
	defer cache.Close()

	// 创建RistrettoStore实例
	ristrettoStore := NewRistrettoStore(cache)

	// 创建StoreTester实例
	tester := store.NewStoreTester(ristrettoStore)

	// 运行测试
	t.Run("Get", tester.TestGet)
	t.Run("MGet", tester.TestMGet)
	t.Run("Exists", tester.TestExists)
	t.Run("MSet", tester.TestMSet)
	t.Run("Del", tester.TestDel)
}

// 测试TTL功能
func TestRistrettoStoreTTL(t *testing.T) {
	// 创建Ristretto缓存实例
	cache, err := ristretto.NewCache(&ristretto.Config[string, interface{}]{
		NumCounters: 1000,
		MaxCost:     100,
		BufferItems: 64,
	})
	if err != nil {
		t.Fatalf("Failed to create Ristretto cache: %v", err)
	}
	defer cache.Close()

	// 创建RistrettoStore实例
	ristrettoStore := NewRistrettoStore(cache)

	ctx := context.Background()
	items := map[string]interface{}{
		"key1": "value1",
	}
	ttl := 1 * time.Second

	// 设置值并设置TTL
	if err := ristrettoStore.MSet(ctx, items, ttl); err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 等待TTL过期
	time.Sleep(1 * time.Second)

	// 验证值已过期
	var result string
	found, err := ristrettoStore.Get(ctx, "key1", &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if found {
		t.Fatalf("Get: expected key1 to be expired")
	}
}