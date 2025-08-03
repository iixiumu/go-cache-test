package store

import (
	"context"
	"reflect"
	"testing"
	"time"
)

// StoreTestSuite 为Store接口提供统一的测试套件
type StoreTestSuite struct {
	store Store
}

// NewStoreTestSuite 创建新的测试套件
func NewStoreTestSuite(store Store) *StoreTestSuite {
	return &StoreTestSuite{store: store}
}

// RunAllTests 运行所有测试
func (s *StoreTestSuite) RunAllTests(t *testing.T) {
	t.Run("TestBasicGetSet", s.TestBasicGetSet)
	t.Run("TestMGetMSet", s.TestMGetMSet)
	t.Run("TestExists", s.TestExists)
	t.Run("TestDel", s.TestDel)
	t.Run("TestTTL", s.TestTTL)
	t.Run("TestComplexTypes", s.TestComplexTypes)
	t.Run("TestNotFound", s.TestNotFound)
}

// TestBasicGetSet 测试基本的Get和Set操作
func (s *StoreTestSuite) TestBasicGetSet(t *testing.T) {
	ctx := context.Background()

	// 测试字符串
	key1 := "test:string"
	value1 := "hello world"

	err := s.store.MSet(ctx, map[string]interface{}{key1: value1}, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	var result1 string
	found, err := s.store.Get(ctx, key1, &result1)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Fatal("Expected to find key")
	}
	if result1 != value1 {
		t.Fatalf("Expected %q, got %q", value1, result1)
	}

	// 测试整数
	key2 := "test:int"
	value2 := 42

	err = s.store.MSet(ctx, map[string]interface{}{key2: value2}, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	var result2 int
	found, err = s.store.Get(ctx, key2, &result2)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Fatal("Expected to find key")
	}
	if result2 != value2 {
		t.Fatalf("Expected %d, got %d", value2, result2)
	}
}

// TestMGetMSet 测试批量操作
func (s *StoreTestSuite) TestMGetMSet(t *testing.T) {
	ctx := context.Background()

	// 设置多个值
	items := map[string]interface{}{
		"batch:1": "value1",
		"batch:2": 123,
		"batch:3": true,
		"batch:4": 3.14,
	}

	err := s.store.MSet(ctx, items, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 批量获取
	var resultMap map[string]interface{}
	err = s.store.MGet(ctx, []string{"batch:1", "batch:2", "batch:3", "batch:4"}, &resultMap)
	if err != nil {
		t.Fatalf("MGet failed: %v", err)
	}

	// 验证结果
	if len(resultMap) != 4 {
		t.Fatalf("Expected 4 items, got %d", len(resultMap))
	}

	if resultMap["batch:1"] != "value1" {
		t.Fatalf("Expected 'value1', got %v", resultMap["batch:1"])
	}
	// 对于Ristretto，数字保持原始类型，对于Redis，JSON会转换为float64
	if resultMap["batch:2"] != 123 && resultMap["batch:2"] != float64(123) {
		t.Fatalf("Expected 123, got %v", resultMap["batch:2"])
	}
	if resultMap["batch:3"] != true {
		t.Fatalf("Expected true, got %v", resultMap["batch:3"])
	}
	if resultMap["batch:4"] != 3.14 {
		t.Fatalf("Expected 3.14, got %v", resultMap["batch:4"])
	}
}

// TestExists 测试键存在性检查
func (s *StoreTestSuite) TestExists(t *testing.T) {
	ctx := context.Background()

	// 设置一些键
	err := s.store.MSet(ctx, map[string]interface{}{
		"exists:1": "value1",
		"exists:2": "value2",
	}, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 检查存在性
	keys := []string{"exists:1", "exists:2", "exists:3"}
	exists, err := s.store.Exists(ctx, keys)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}

	expected := map[string]bool{
		"exists:1": true,
		"exists:2": true,
		"exists:3": false,
	}

	if !reflect.DeepEqual(exists, expected) {
		t.Fatalf("Expected %v, got %v", expected, exists)
	}
}

// TestDel 测试删除操作
func (s *StoreTestSuite) TestDel(t *testing.T) {
	ctx := context.Background()

	// 设置一些键
	err := s.store.MSet(ctx, map[string]interface{}{
		"del:1": "value1",
		"del:2": "value2",
		"del:3": "value3",
	}, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 删除键
	deleted, err := s.store.Del(ctx, "del:1", "del:2")
	if err != nil {
		t.Fatalf("Del failed: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("Expected 2 deleted, got %d", deleted)
	}

	// 验证删除结果
	exists, err := s.store.Exists(ctx, []string{"del:1", "del:2", "del:3"})
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}

	expected := map[string]bool{
		"del:1": false,
		"del:2": false,
		"del:3": true,
	}

	if !reflect.DeepEqual(exists, expected) {
		t.Fatalf("Expected %v, got %v", expected, exists)
	}
}

// TestTTL 测试TTL功能
func (s *StoreTestSuite) TestTTL(t *testing.T) {
	ctx := context.Background()

	// 设置带TTL的键
	err := s.store.MSet(ctx, map[string]interface{}{
		"ttl:test": "value",
	}, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// 立即获取应该存在
	var result string
	found, err := s.store.Get(ctx, "ttl:test", &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Fatal("Expected to find key immediately")
	}

	// 等待TTL过期
	time.Sleep(200 * time.Millisecond)

	// 再次获取应该不存在
	found, err = s.store.Get(ctx, "ttl:test", &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if found {
		t.Fatal("Expected key to be expired")
	}
}

// TestComplexTypes 测试复杂类型
func (s *StoreTestSuite) TestComplexTypes(t *testing.T) {
	ctx := context.Background()

	type Person struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	person := Person{Name: "Alice", Age: 30}

	err := s.store.MSet(ctx, map[string]interface{}{"person": person}, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	var result Person
	found, err := s.store.Get(ctx, "person", &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Fatal("Expected to find key")
	}
	if result.Name != person.Name || result.Age != person.Age {
		t.Fatalf("Expected %+v, got %+v", person, result)
	}

	// 测试切片
	slice := []int{1, 2, 3, 4, 5}
	err = s.store.MSet(ctx, map[string]interface{}{"slice": slice}, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	var resultSlice []int
	found, err = s.store.Get(ctx, "slice", &resultSlice)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Fatal("Expected to find key")
	}
	if !reflect.DeepEqual(resultSlice, slice) {
		t.Fatalf("Expected %v, got %v", slice, resultSlice)
	}
}

// TestNotFound 测试键不存在的情况
func (s *StoreTestSuite) TestNotFound(t *testing.T) {
	ctx := context.Background()

	var result string
	found, err := s.store.Get(ctx, "nonexistent", &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if found {
		t.Fatal("Expected key to not exist")
	}

	// 批量获取不存在的键
	var resultMap map[string]interface{}
	err = s.store.MGet(ctx, []string{"nonexistent1", "nonexistent2"}, &resultMap)
	if err != nil {
		t.Fatalf("MGet failed: %v", err)
	}
	if len(resultMap) != 0 {
		t.Fatalf("Expected empty map, got %v", resultMap)
	}
}
