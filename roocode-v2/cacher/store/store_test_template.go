package store

import (
	"context"
	"testing"
	"time"
)

// StoreTester 是Store接口的统一测试模板
type StoreTester struct {
	NewStore func() Store
}

// RunAllTests 运行所有Store接口测试
func (st *StoreTester) RunAllTests(t *testing.T) {
	t.Run("TestGetSet", st.testGetSet)
	t.Run("TestMGet", st.testMGet)
	t.Run("TestExists", st.testExists)
	t.Run("TestMSet", st.testMSet)
	t.Run("TestDel", st.testDel)
	t.Run("TestTTL", st.testTTL)
}

func (st *StoreTester) testGetSet(t *testing.T) {
	store := st.NewStore()
	ctx := context.Background()

	// 测试设置和获取字符串
	key := "test_key"
	value := "test_value"

	// 设置值
	items := map[string]interface{}{key: value}
	err := store.MSet(ctx, items, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 获取值
	var result string
	found, err := store.Get(ctx, key, &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Fatalf("Key not found")
	}
	if result != value {
		t.Fatalf("Expected %s, got %s", value, result)
	}

	// 测试不存在的键
	var result2 string
	found, err = store.Get(ctx, "nonexistent", &result2)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if found {
		t.Fatalf("Expected key not found")
	}
}

func (st *StoreTester) testMGet(t *testing.T) {
	store := st.NewStore()
	ctx := context.Background()

	// 设置多个值
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	err := store.MSet(ctx, items, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 批量获取
	keys := []string{"key1", "key2", "key3", "nonexistent"}
	result := make(map[string]string)
	err = store.MGet(ctx, keys, &result)
	if err != nil {
		t.Fatalf("MGet failed: %v", err)
	}

	// 验证结果
	if len(result) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(result))
	}
	if result["key1"] != "value1" {
		t.Fatalf("Expected value1, got %s", result["key1"])
	}
	if result["key2"] != "value2" {
		t.Fatalf("Expected value2, got %s", result["key2"])
	}
	if result["key3"] != "value3" {
		t.Fatalf("Expected value3, got %s", result["key3"])
	}
}

func (st *StoreTester) testExists(t *testing.T) {
	store := st.NewStore()
	ctx := context.Background()

	// 设置值
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	err := store.MSet(ctx, items, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 检查存在性
	keys := []string{"key1", "key2", "nonexistent"}
	exists, err := store.Exists(ctx, keys)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}

	// 验证结果
	if len(exists) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(exists))
	}
	if !exists["key1"] {
		t.Fatalf("Expected key1 to exist")
	}
	if !exists["key2"] {
		t.Fatalf("Expected key2 to exist")
	}
	if exists["nonexistent"] {
		t.Fatalf("Expected nonexistent key to not exist")
	}
}

func (st *StoreTester) testMSet(t *testing.T) {
	store := st.NewStore()
	ctx := context.Background()

	// 批量设置
	items := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}
	err := store.MSet(ctx, items, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 验证设置的值
	var result1 string
	found, err := store.Get(ctx, "key1", &result1)
	if err != nil || !found || result1 != "value1" {
		t.Fatalf("Failed to get key1")
	}

	var result2 int
	found, err = store.Get(ctx, "key2", &result2)
	if err != nil || !found || result2 != 42 {
		t.Fatalf("Failed to get key2")
	}

	var result3 bool
	found, err = store.Get(ctx, "key3", &result3)
	if err != nil || !found || !result3 {
		t.Fatalf("Failed to get key3")
	}
}

func (st *StoreTester) testDel(t *testing.T) {
	store := st.NewStore()
	ctx := context.Background()

	// 设置值
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	err := store.MSet(ctx, items, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 删除部分键
	keys := []string{"key1", "key2", "nonexistent"}
	deleted, err := store.Del(ctx, keys...)
	if err != nil {
		t.Fatalf("Del failed: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("Expected 2 deleted keys, got %d", deleted)
	}

	// 验证键已被删除
	var result string
	found, err := store.Get(ctx, "key1", &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if found {
		t.Fatalf("Expected key1 to be deleted")
	}

	found, err = store.Get(ctx, "key2", &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if found {
		t.Fatalf("Expected key2 to be deleted")
	}

	// 验证未删除的键仍然存在
	found, err = store.Get(ctx, "key3", &result)
	if err != nil || !found || result != "value3" {
		t.Fatalf("Expected key3 to still exist")
	}
}

func (st *StoreTester) testTTL(t *testing.T) {
	store := st.NewStore()
	ctx := context.Background()

	// 设置带TTL的值
	key := "ttl_key"
	value := "ttl_value"
	items := map[string]interface{}{key: value}
	err := store.MSet(ctx, items, time.Second)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 验证值存在
	var result string
	found, err := store.Get(ctx, key, &result)
	if err != nil || !found || result != value {
		t.Fatalf("Failed to get ttl_key")
	}

	// 等待过期
	time.Sleep(time.Second * 2)

	// 验证值已过期
	found, err = store.Get(ctx, key, &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if found {
		t.Fatalf("Expected ttl_key to be expired")
	}
}
