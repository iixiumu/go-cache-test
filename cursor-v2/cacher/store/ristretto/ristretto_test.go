package ristretto

import (
	"testing"

	"go-cache/cacher/store"
)

func TestRistrettoStore(t *testing.T) {
	// 创建Ristretto Store
	ristrettoStore, err := NewRistrettoStore()
	if err != nil {
		t.Fatalf("Failed to create Ristretto store: %v", err)
	}

	// 创建测试套件并运行所有测试
	testSuite := store.NewStoreTestSuite(ristrettoStore)
	testSuite.RunAllTests(t)
}
