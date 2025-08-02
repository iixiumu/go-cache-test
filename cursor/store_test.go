package cache

import (
	"context"
	"testing"
	"time"
)

func TestMockStore_Get(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	// 测试获取存在的键
	store.(*MockStore).data["key1"] = "value1"
	var result string
	found, err := store.Get(ctx, "key1", &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Fatal("Expected to find key1")
	}
	if result != "value1" {
		t.Fatalf("Expected 'value1', got '%s'", result)
	}

	// 测试获取不存在的键
	var result2 string
	found, err = store.Get(ctx, "nonexistent", &result2)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if found {
		t.Fatal("Expected not to find nonexistent key")
	}
}

func TestMockStore_MGet(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	// 设置一些数据
	store.(*MockStore).data["key1"] = "value1"
	store.(*MockStore).data["key2"] = "value2"
	store.(*MockStore).data["key3"] = "value3"

	// 测试批量获取
	result := make(map[string]string)
	err := store.MGet(ctx, []string{"key1", "key2", "key4"}, &result)
	if err != nil {
		t.Fatalf("MGet failed: %v", err)
	}

	// 验证结果
	expected := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	for key, expectedValue := range expected {
		if actualValue, exists := result[key]; !exists {
			t.Fatalf("Expected key '%s' to exist", key)
		} else if actualValue != expectedValue {
			t.Fatalf("Expected '%s' for key '%s', got '%s'", expectedValue, key, actualValue)
		}
	}

	// 验证不存在的键不在结果中
	if _, exists := result["key4"]; exists {
		t.Fatal("Expected key4 not to exist in result")
	}
}

func TestMockStore_Exists(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	// 设置一些数据
	store.(*MockStore).data["key1"] = "value1"
	store.(*MockStore).data["key2"] = "value2"

	// 测试批量检查存在性
	result, err := store.Exists(ctx, []string{"key1", "key2", "key3"})
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}

	expected := map[string]bool{
		"key1": true,
		"key2": true,
		"key3": false,
	}

	for key, expectedValue := range expected {
		if actualValue, exists := result[key]; !exists {
			t.Fatalf("Expected key '%s' in result", key)
		} else if actualValue != expectedValue {
			t.Fatalf("Expected %v for key '%s', got %v", expectedValue, key, actualValue)
		}
	}
}

func TestMockStore_MSet(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	// 测试批量设置
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	err := store.MSet(ctx, items, time.Hour)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 验证设置结果
	for key, expectedValue := range items {
		if actualValue, exists := store.(*MockStore).data[key]; !exists {
			t.Fatalf("Expected key '%s' to exist", key)
		} else if actualValue != expectedValue {
			t.Fatalf("Expected '%v' for key '%s', got '%v'", expectedValue, key, actualValue)
		}
	}
}

func TestMockStore_Del(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	// 设置一些数据
	store.(*MockStore).data["key1"] = "value1"
	store.(*MockStore).data["key2"] = "value2"
	store.(*MockStore).data["key3"] = "value3"

	// 测试删除
	deleted, err := store.Del(ctx, "key1", "key2", "key4")
	if err != nil {
		t.Fatalf("Del failed: %v", err)
	}

	if deleted != 2 {
		t.Fatalf("Expected to delete 2 keys, deleted %d", deleted)
	}

	// 验证删除结果
	if _, exists := store.(*MockStore).data["key1"]; exists {
		t.Fatal("Expected key1 to be deleted")
	}
	if _, exists := store.(*MockStore).data["key2"]; exists {
		t.Fatal("Expected key2 to be deleted")
	}
	if _, exists := store.(*MockStore).data["key3"]; !exists {
		t.Fatal("Expected key3 to still exist")
	}
}

func TestMockStore_EmptyKeys(t *testing.T) {
	store := NewMockStore()
	ctx := context.Background()

	// 测试空键列表
	result := make(map[string]string)
	err := store.MGet(ctx, []string{}, &result)
	if err != nil {
		t.Fatalf("MGet with empty keys failed: %v", err)
	}

	exists, err := store.Exists(ctx, []string{})
	if err != nil {
		t.Fatalf("Exists with empty keys failed: %v", err)
	}
	if len(exists) != 0 {
		t.Fatal("Expected empty result for empty keys")
	}

	deleted, err := store.Del(ctx)
	if err != nil {
		t.Fatalf("Del with empty keys failed: %v", err)
	}
	if deleted != 0 {
		t.Fatalf("Expected to delete 0 keys, deleted %d", deleted)
	}
}
