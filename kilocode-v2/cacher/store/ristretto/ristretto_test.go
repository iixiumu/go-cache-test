package ristretto

import (
	"context"
	"testing"
)

func TestRistrettoStore(t *testing.T) {
	// 创建RistrettoStore实例
	store, err := NewRistrettoStore()
	if err != nil {
		t.Fatalf("Failed to create RistrettoStore: %v", err)
	}

	// 这里可以添加一些基本的测试
	// 由于我们没有统一的测试运行器，我们只做简单的测试

	// 测试MSet和Get
	ctx := context.Background()
	items := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}

	err = store.MSet(ctx, items, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 测试Get
	var result string
	found, err := store.Get(ctx, "key1", &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if !found {
		t.Error("Key should be found")
	}

	if result != "value1" {
		t.Errorf("Expected value1, got %v", result)
	}

	// 测试Get with int
	var intResult int
	found, err = store.Get(ctx, "key2", &intResult)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if !found {
		t.Error("Key should be found")
	}

	if intResult != 42 {
		t.Errorf("Expected 42, got %v", intResult)
	}
}
