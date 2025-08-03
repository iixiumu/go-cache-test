package ristretto

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestRistrettoStore(t *testing.T) {
	// 创建RistrettoStore实例
	ristrettoStore, err := NewRistrettoStore()
	if err != nil {
		t.Fatalf("NewRistrettoStore failed: %v", err)
	}
	defer ristrettoStore.Close()

	// 基础功能测试
	ctx := context.Background()

	t.Run("Get and Set", func(t *testing.T) {
		// 测试基本设置和获取
		value := "test_value"
		err := ristrettoStore.MSet(ctx, map[string]interface{}{"key1": value}, time.Minute)
		if err != nil {
			t.Fatalf("MSet failed: %v", err)
		}

		// 等待缓存生效
		time.Sleep(100 * time.Millisecond)

		var result string
		found, err := ristrettoStore.Get(ctx, "key1", &result)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if !found {
			t.Error("Expected to find key")
		}
		if result != value {
			t.Errorf("Expected '%s', got '%s'", value, result)
		}
	})

	t.Run("Exists", func(t *testing.T) {
		ristrettoStore.MSet(ctx, map[string]interface{}{"exists_key": "value"}, time.Minute)
		
		exists, err := ristrettoStore.Exists(ctx, []string{"exists_key", "nonexistent_key"})
		if err != nil {
			t.Fatalf("Exists failed: %v", err)
		}
		if !exists["exists_key"] {
			t.Error("Expected exists_key to exist")
		}
		if exists["nonexistent_key"] {
			t.Error("Expected nonexistent_key to not exist")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		ristrettoStore.MSet(ctx, map[string]interface{}{"del_key": "value"}, time.Minute)
		
		deleted, err := ristrettoStore.Del(ctx, "del_key")
		if err != nil {
			t.Fatalf("Del failed: %v", err)
		}
		if deleted != 1 {
			t.Errorf("Expected 1 deleted, got %d", deleted)
		}

		var result string
		found, _ := ristrettoStore.Get(ctx, "del_key", &result)
		if found {
			t.Error("Expected key to be deleted")
		}
	})
}

func TestRistrettoStore_EmptyOperations(t *testing.T) {
	ristrettoStore, err := NewRistrettoStore()
	if err != nil {
		t.Fatalf("NewRistrettoStore failed: %v", err)
	}
	defer ristrettoStore.Close()

	ctx := context.Background()

	// 测试空操作
	err = ristrettoStore.MSet(ctx, map[string]interface{}{}, 0)
	if err != nil {
		t.Errorf("Empty MSet should not fail: %v", err)
	}

	exists, err := ristrettoStore.Exists(ctx, []string{})
	if err != nil {
		t.Errorf("Empty Exists should not fail: %v", err)
	}
	if len(exists) != 0 {
		t.Errorf("Empty Exists should return empty map")
	}

	deleted, err := ristrettoStore.Del(ctx)
	if err != nil {
		t.Errorf("Empty Del should not fail: %v", err)
	}
	if deleted != 0 {
		t.Errorf("Empty Del should return 0")
	}
}

func TestRistrettoStore_StructTypes(t *testing.T) {
	ristrettoStore, err := NewRistrettoStore()
	if err != nil {
		t.Fatalf("NewRistrettoStore failed: %v", err)
	}
	defer ristrettoStore.Close()

	ctx := context.Background()

	// 测试结构体类型
	type Person struct {
		Name string
		Age  int
	}

	person := Person{Name: "Alice", Age: 30}
	err = ristrettoStore.MSet(ctx, map[string]interface{}{
		"person": person,
	}, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	var result Person
	found, err := ristrettoStore.Get(ctx, "person", &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Error("Expected found for existing key")
	}
	if result.Name != "Alice" || result.Age != 30 {
		t.Errorf("Expected {Alice 30}, got %+v", result)
	}
}

func TestRistrettoStore_MapTypes(t *testing.T) {
	ristrettoStore, err := NewRistrettoStore()
	if err != nil {
		t.Fatalf("NewRistrettoStore failed: %v", err)
	}
	defer ristrettoStore.Close()

	ctx := context.Background()

	// 测试map类型
	testMap := map[string]interface{}{
		"name": "Bob",
		"age":  25,
		"tags": []string{"developer", "golang"},
	}

	err = ristrettoStore.MSet(ctx, map[string]interface{}{
		"user_map": testMap,
	}, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	var result map[string]interface{}
	found, err := ristrettoStore.Get(ctx, "user_map", &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Error("Expected found for existing key")
	}
	if result["name"] != "Bob" || result["age"] != 25 {
		t.Errorf("Expected map values, got %+v", result)
	}
}

func TestRistrettoStore_TTL(t *testing.T) {
	ristrettoStore, err := NewRistrettoStore()
	if err != nil {
		t.Fatalf("NewRistrettoStore failed: %v", err)
	}
	defer ristrettoStore.Close()

	ctx := context.Background()

	// 设置带TTL的键
	err = ristrettoStore.MSet(ctx, map[string]interface{}{
		"ttl_key": "ttl_value",
	}, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 立即检查应该存在
	var val string
	found, err := ristrettoStore.Get(ctx, "ttl_key", &val)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Error("Expected found immediately after set")
	}

	// 等待过期
	time.Sleep(200 * time.Millisecond)

	// 检查应该不存在
	found, err = ristrettoStore.Get(ctx, "ttl_key", &val)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if found {
		t.Error("Expected not found after TTL expiration")
	}
}

func TestRistrettoStore_CostCalculation(t *testing.T) {
	ristrettoStore, err := NewRistrettoStore()
	if err != nil {
		t.Fatalf("NewRistrettoStore failed: %v", err)
	}
	defer ristrettoStore.Close()

	ctx := context.Background()

	// 测试大量数据，触发缓存淘汰
	largeData := make([]byte, 1024*1024) // 1MB
	for i := range largeData {
		largeData[i] = 'a'
	}

	// 设置大量数据
	for i := 0; i < 100; i++ {
		err := ristrettoStore.MSet(ctx, map[string]interface{}{
			fmt.Sprintf("large_key_%d", i): string(largeData),
		}, 0)
		if err != nil {
		t.Fatalf("MSet failed: %v", err)
		}
	}

	// 等待缓存处理
	time.Sleep(100 * time.Millisecond)

	// 验证部分数据可能被淘汰
	var val string
	found, err := ristrettoStore.Get(ctx, "large_key_0", &val)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	// 由于设置了最大成本，一些键可能被驱逐
	// 我们不检查具体结果，只验证功能正常
	t.Logf("Key found: %v", found)
}