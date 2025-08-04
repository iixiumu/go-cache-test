package ristretto

import (
	"testing"

	"go-cache/cacher/store"

	"github.com/dgraph-io/ristretto/v2"
)

func TestRistrettoStore(t *testing.T) {
	// 创建Ristretto Store
	cache, err := ristretto.NewCache(&ristretto.Config[string, interface{}]{
		NumCounters: 1e7,     // 键跟踪数
		MaxCost:     1 << 30, // 最大缓存成本(1GB)
		BufferItems: 64,      // 缓冲区大小
	})
	if err != nil {
		t.Fatalf("Failed to create Ristretto store: %v", err)
	}
	ristrettoStore, err := NewRistrettoStore(cache)
	if err != nil {
		t.Fatalf("Failed to create Ristretto store: %v", err)
	}

	// 创建测试套件并运行所有测试
	testSuite := store.NewStoreTestSuite(ristrettoStore)
	testSuite.RunAllTests(t)
}
