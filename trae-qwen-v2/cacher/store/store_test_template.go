package store

import (
	"context"
	"testing"
	"time"
)

// StoreTester 用于测试Store接口实现的测试模板
type StoreTester struct {
	store Store
}

// NewStoreTester 创建新的Store测试器
func NewStoreTester(store Store) *StoreTester {
	return &StoreTester{store: store}
}

// TestGet 测试Get方法
func (st *StoreTester) TestGet(t *testing.T) {
	ctx := context.Background()
	key := "test_key"
	value := "test_value"

	// 设置值
	items := map[string]interface{}{key: value}
	ttl := 10 * time.Second
	if err := st.store.MSet(ctx, items, ttl); err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 获取值
	var result string
	found, err := st.store.Get(ctx, key, &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Fatalf("Get: key not found")
	}
	if result != value {
		t.Fatalf("Get: expected %v, got %v", value, result)
	}
}

// TestMGet 测试MGet方法
func (st *StoreTester) TestMGet(t *testing.T) {
	ctx := context.Background()
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	ttl := 10 * time.Second

	// 设置值
	if err := st.store.MSet(ctx, items, ttl); err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 批量获取值
	keys := []string{"key1", "key2", "key3"}
	result := make(map[string]string)
	if err := st.store.MGet(ctx, keys, &result); err != nil {
		t.Fatalf("MGet failed: %v", err)
	}

	// 验证结果
	for k, v := range items {
		if result[k] != v {
			t.Fatalf("MGet: expected %v for key %s, got %v", v, k, result[k])
		}
	}
}

// TestExists 测试Exists方法
func (st *StoreTester) TestExists(t *testing.T) {
	ctx := context.Background()
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	ttl := 10 * time.Second

	// 设置值
	if err := st.store.MSet(ctx, items, ttl); err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 检查存在性
	keys := []string{"key1", "key2", "key3"}
	exists, err := st.store.Exists(ctx, keys)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}

	// 验证结果
	if !exists["key1"] || !exists["key2"] {
		t.Fatalf("Exists: expected key1 and key2 to exist")
	}
	if exists["key3"] {
		t.Fatalf("Exists: expected key3 to not exist")
	}
}

// TestMSet 测试MSet方法
func (st *StoreTester) TestMSet(t *testing.T) {
	ctx := context.Background()
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	ttl := 10 * time.Second

	// 设置值
	if err := st.store.MSet(ctx, items, ttl); err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 验证值已设置
	var result1, result2 string
	found1, err1 := st.store.Get(ctx, "key1", &result1)
	found2, err2 := st.store.Get(ctx, "key2", &result2)

	if err1 != nil || err2 != nil {
		t.Fatalf("Get failed after MSet: %v, %v", err1, err2)
	}
	if !found1 || !found2 {
		t.Fatalf("Get: keys not found after MSet")
	}
	if result1 != "value1" || result2 != "value2" {
		t.Fatalf("Get: unexpected values after MSet")
	}
}

// TestDel 测试Del方法
func (st *StoreTester) TestDel(t *testing.T) {
	ctx := context.Background()
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	ttl := 10 * time.Second

	// 设置值
	if err := st.store.MSet(ctx, items, ttl); err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 删除值
	deleted, err := st.store.Del(ctx, "key1", "key2", "key3")
	if err != nil {
		t.Fatalf("Del failed: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("Del: expected 2 keys deleted, got %d", deleted)
	}

	// 验证值已删除
	var result string
	found, err := st.store.Get(ctx, "key1", &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if found {
		t.Fatalf("Get: expected key1 to be deleted")
	}
}