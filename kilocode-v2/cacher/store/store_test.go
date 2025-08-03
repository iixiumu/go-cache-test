package store

import (
	"context"
	"testing"
	"time"
)

// StoreTester 定义Store接口测试的通用接口
type StoreTester interface {
	// NewStore 创建一个新的Store实例用于测试
	NewStore() (Store, error)

	// Name 返回Store实现的名称，用于测试报告
	Name() string

	// SetupTest 在每个测试前执行的设置操作
	SetupTest(t *testing.T)

	// TeardownTest 在每个测试后执行的清理操作
	TeardownTest(t *testing.T)
}

// RunStoreTests 运行所有Store接口的测试
func RunStoreTests(t *testing.T, tester StoreTester) {
	tests := []struct {
		name string
		fn   func(t *testing.T, store Store)
	}{
		{"TestGetSet", testGetSet},
		{"TestMGet", testMGet},
		{"TestExists", testExists},
		{"TestMSet", testMSet},
		{"TestDel", testDel},
		{"TestGetNonExistent", testGetNonExistent},
		{"TestMGetPartial", testMGetPartial},
		{"TestMSetWithTTL", testMSetWithTTL},
	}

	for _, tt := range tests {
		t.Run(tester.Name()+"/"+tt.name, func(t *testing.T) {
			tester.SetupTest(t)
			defer tester.TeardownTest(t)

			store, err := tester.NewStore()
			if err != nil {
				t.Fatalf("Failed to create store: %v", err)
			}

			tt.fn(t, store)
		})
	}
}

// 测试Get和Set操作
func testGetSet(t *testing.T, store Store) {
	ctx := context.Background()

	// 设置测试数据
	key := "test_key"
	value := "test_value"

	err := store.MSet(ctx, map[string]interface{}{key: value}, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 获取数据
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
}

// 测试MGet操作
func testMGet(t *testing.T, store Store) {
	ctx := context.Background()

	// 设置测试数据
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	err := store.MSet(ctx, items, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 批量获取数据
	keys := []string{"key1", "key2", "key3"}
	result := make(map[string]string)
	err = store.MGet(ctx, keys, &result)
	if err != nil {
		t.Fatalf("MGet failed: %v", err)
	}

	if len(result) != len(items) {
		t.Errorf("Expected %d items, got %d", len(items), len(result))
	}

	for k, v := range items {
		if result[k] != v {
			t.Errorf("Key %s: expected %v, got %v", k, v, result[k])
		}
	}
}

// 测试Exists操作
func testExists(t *testing.T, store Store) {
	ctx := context.Background()

	// 设置测试数据
	items := map[string]interface{}{
		"exist_key1": "value1",
		"exist_key2": "value2",
	}

	err := store.MSet(ctx, items, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 检查存在性
	keys := []string{"exist_key1", "exist_key2", "non_exist_key"}
	exists, err := store.Exists(ctx, keys)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}

	if len(exists) != 3 {
		t.Errorf("Expected 3 results, got %d", len(exists))
	}

	if !exists["exist_key1"] {
		t.Error("exist_key1 should exist")
	}

	if !exists["exist_key2"] {
		t.Error("exist_key2 should exist")
	}

	if exists["non_exist_key"] {
		t.Error("non_exist_key should not exist")
	}
}

// 测试MSet操作
func testMSet(t *testing.T, store Store) {
	ctx := context.Background()

	// 批量设置数据
	items := map[string]interface{}{
		"mset_key1": "value1",
		"mset_key2": 42,
		"mset_key3": true,
	}

	err := store.MSet(ctx, items, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 验证数据是否正确设置
	var result1 string
	found, err := store.Get(ctx, "mset_key1", &result1)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if !found {
		t.Error("mset_key1 should be found")
	}

	if result1 != "value1" {
		t.Errorf("Expected value1, got %v", result1)
	}

	var result2 int
	found, err = store.Get(ctx, "mset_key2", &result2)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if !found {
		t.Error("mset_key2 should be found")
	}

	if result2 != 42 {
		t.Errorf("Expected 42, got %v", result2)
	}

	var result3 bool
	found, err = store.Get(ctx, "mset_key3", &result3)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if !found {
		t.Error("mset_key3 should be found")
	}

	if !result3 {
		t.Errorf("Expected true, got %v", result3)
	}
}

// 测试Del操作
func testDel(t *testing.T, store Store) {
	ctx := context.Background()

	// 设置测试数据
	items := map[string]interface{}{
		"del_key1": "value1",
		"del_key2": "value2",
		"del_key3": "value3",
	}

	err := store.MSet(ctx, items, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 删除部分键
	deleted, err := store.Del(ctx, "del_key1", "del_key2")
	if err != nil {
		t.Fatalf("Del failed: %v", err)
	}

	if deleted != 2 {
		t.Errorf("Expected 2 deleted keys, got %d", deleted)
	}

	// 验证键是否被删除
	found, err := store.Get(ctx, "del_key1", new(string))
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if found {
		t.Error("del_key1 should not be found")
	}

	found, err = store.Get(ctx, "del_key2", new(string))
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if found {
		t.Error("del_key2 should not be found")
	}

	// 验证未删除的键仍然存在
	var result string
	found, err = store.Get(ctx, "del_key3", &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if !found {
		t.Error("del_key3 should be found")
	}

	if result != "value3" {
		t.Errorf("Expected value3, got %v", result)
	}
}

// 测试获取不存在的键
func testGetNonExistent(t *testing.T, store Store) {
	ctx := context.Background()

	var result string
	found, err := store.Get(ctx, "non_existent_key", &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if found {
		t.Error("Non-existent key should not be found")
	}
}

// 测试部分命中MGet
func testMGetPartial(t *testing.T, store Store) {
	ctx := context.Background()

	// 设置部分测试数据
	items := map[string]interface{}{
		"partial_key1": "value1",
		// partial_key2 不存在
		"partial_key3": "value3",
	}

	err := store.MSet(ctx, items, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 批量获取数据，包含存在的和不存在的键
	keys := []string{"partial_key1", "partial_key2", "partial_key3"}
	result := make(map[string]string)
	err = store.MGet(ctx, keys, &result)
	if err != nil {
		t.Fatalf("MGet failed: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 items, got %d", len(result))
	}

	if result["partial_key1"] != "value1" {
		t.Errorf("Expected value1, got %v", result["partial_key1"])
	}

	if result["partial_key3"] != "value3" {
		t.Errorf("Expected value3, got %v", result["partial_key3"])
	}

	// 不应该包含不存在的键
	if _, exists := result["partial_key2"]; exists {
		t.Error("partial_key2 should not be in result")
	}
}

// 测试带TTL的MSet
func testMSetWithTTL(t *testing.T, store Store) {
	ctx := context.Background()

	// 批量设置带TTL的数据
	items := map[string]interface{}{
		"ttl_key1": "value1",
		"ttl_key2": "value2",
	}

	ttl := 100 * time.Millisecond
	err := store.MSet(ctx, items, ttl)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 立即获取数据，应该能获取到
	var result1 string
	found, err := store.Get(ctx, "ttl_key1", &result1)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if !found {
		t.Error("ttl_key1 should be found immediately")
	}

	if result1 != "value1" {
		t.Errorf("Expected value1, got %v", result1)
	}

	// 等待TTL过期
	time.Sleep(ttl + 10*time.Millisecond)

	// 再次获取数据，应该获取不到
	found, err = store.Get(ctx, "ttl_key1", &result1)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if found {
		t.Error("ttl_key1 should not be found after TTL expired")
	}
}
