package store

import (
	"context"
	"testing"
	"time"
)

// StoreTester Store接口统一测试模板
type StoreTester interface {
	// NewStore 创建一个新的Store实例用于测试
	NewStore() (Store, error)
	// Name 返回Store实现的名称，用于测试报告
	Name() string
	// SetupTest 在每个测试前执行的设置
	SetupTest(t *testing.T)
	// TeardownTest 在每个测试后执行的清理
	TeardownTest(t *testing.T)
}

// RunStoreTests 运行Store接口的统一测试
func RunStoreTests(t *testing.T, tester StoreTester) {
	tests := []struct {
		name string
		fn   func(t *testing.T, store Store)
	}{
		{"TestGetSet", testGetSet},
		{"TestMGetMSet", testMGetMSet},
		{"TestExists", testExists},
		{"TestDel", testDel},
		{"TestTTL", testTTL},
		{"TestMGetPartial", testMGetPartial},
	}

	for _, tt := range tests {
		t.Run(tester.Name()+"/"+tt.name, func(t *testing.T) {
			tester.SetupTest(t)
			defer tester.TeardownTest(t)

			store, err := tester.NewStore()
			if err != nil {
				t.Fatalf("failed to create store: %v", err)
			}

			tt.fn(t, store)
		})
	}
}

func testGetSet(t *testing.T, store Store) {
	ctx := context.Background()

	// Test string value
	key := "test_key"
	value := "test_value"

	// Set value
	err := store.MSet(ctx, map[string]interface{}{key: value}, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// Get value
	var result string
	found, err := store.Get(ctx, key, &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if !found {
		t.Error("Key should be found")
	}

	if result != value {
		t.Errorf("Expected %v, got %v", value, result)
	}

	// Test non-existent key
	var result2 string
	found, err = store.Get(ctx, "non_existent_key", &result2)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if found {
		t.Error("Non-existent key should not be found")
	}
}

func testMGetMSet(t *testing.T, store Store) {
	ctx := context.Background()

	// Set multiple values
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	err := store.MSet(ctx, items, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// Get multiple values
	keys := []string{"key1", "key2", "key3"}
	result := make(map[string]string)
	err = store.MGet(ctx, keys, &result)
	if err != nil {
		t.Fatalf("MGet failed: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 results, got %d", len(result))
	}

	for k, v := range items {
		if result[k] != v {
			t.Errorf("Key %s: expected %v, got %v", k, v, result[k])
		}
	}
}

func testExists(t *testing.T, store Store) {
	ctx := context.Background()

	// Set some values
	items := map[string]interface{}{
		"exist_key1": "value1",
		"exist_key2": "value2",
	}

	err := store.MSet(ctx, items, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// Check existence
	keys := []string{"exist_key1", "exist_key2", "non_existent_key"}
	exists, err := store.Exists(ctx, keys)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}

	if len(exists) != 3 {
		t.Errorf("Expected 3 existence results, got %d", len(exists))
	}

	if !exists["exist_key1"] {
		t.Error("exist_key1 should exist")
	}

	if !exists["exist_key2"] {
		t.Error("exist_key2 should exist")
	}

	if exists["non_existent_key"] {
		t.Error("non_existent_key should not exist")
	}
}

func testDel(t *testing.T, store Store) {
	ctx := context.Background()

	// Set some values
	items := map[string]interface{}{
		"del_key1": "value1",
		"del_key2": "value2",
		"del_key3": "value3",
	}

	err := store.MSet(ctx, items, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// Delete some keys
	deleted, err := store.Del(ctx, "del_key1", "del_key2", "non_existent_key")
	if err != nil {
		t.Fatalf("Del failed: %v", err)
	}

	if deleted != 2 {
		t.Errorf("Expected 2 deletions, got %d", deleted)
	}

	// Verify deletion
	var result string
	found, err := store.Get(ctx, "del_key1", &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if found {
		t.Error("del_key1 should be deleted")
	}
}

func testTTL(t *testing.T, store Store) {
	ctx := context.Background()

	// Set value with TTL
	key := "ttl_key"
	value := "ttl_value"
	ttl := 2 * time.Second

	err := store.MSet(ctx, map[string]interface{}{key: value}, ttl)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// Check if value exists immediately
	var result string
	found, err := store.Get(ctx, key, &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if !found {
		t.Error("Key should exist immediately after setting")
	}

	if result != value {
		t.Errorf("Expected %v, got %v", value, result)
	}

	// Note: Actual TTL testing would require time-based assertions
	// which are implementation-specific and should be tested in
	// each Store implementation's specific tests
}

func testMGetPartial(t *testing.T, store Store) {
	ctx := context.Background()

	// Set only some values
	items := map[string]interface{}{
		"partial_key1": "value1",
		"partial_key3": "value3",
	}

	err := store.MSet(ctx, items, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// Try to get all keys including non-existent ones
	keys := []string{"partial_key1", "partial_key2", "partial_key3"}
	result := make(map[string]string)
	err = store.MGet(ctx, keys, &result)
	if err != nil {
		t.Fatalf("MGet failed: %v", err)
	}

	// Should only get the existing keys
	if len(result) != 2 {
		t.Errorf("Expected 2 results, got %d", len(result))
	}

	if result["partial_key1"] != "value1" {
		t.Errorf("Expected value1, got %v", result["partial_key1"])
	}

	if result["partial_key3"] != "value3" {
		t.Errorf("Expected value3, got %v", result["partial_key3"])
	}

	// partial_key2 should not be in result
	if _, exists := result["partial_key2"]; exists {
		t.Error("partial_key2 should not exist in result")
	}
}
