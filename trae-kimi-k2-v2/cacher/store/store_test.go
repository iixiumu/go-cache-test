package store

import (
	"context"
	"reflect"
	"testing"
	"time"
)

// StoreTestSuite Store接口统一测试套件
type StoreTestSuite struct {
	store Store
	t     *testing.T
}

// NewStoreTestSuite 创建测试套件
func NewStoreTestSuite(store Store, t *testing.T) *StoreTestSuite {
	return &StoreTestSuite{
		store: store,
		t:     t,
	}
}

// Run 运行所有测试
func (s *StoreTestSuite) Run() {
	s.TestGetSet()
	s.TestMGetMSet()
	s.TestExists()
	s.TestDel()
	s.TestTTL()
}

// TestGetSet 测试基本的Get和Set操作
func (s *StoreTestSuite) TestGetSet() {
	ctx := context.Background()
	
	// 测试字符串类型
	var strVal string
	found, err := s.store.Get(ctx, "test_str", &strVal)
	if err != nil {
		s.t.Fatalf("Get failed: %v", err)
	}
	if found {
		s.t.Error("Expected not found for non-existent key")
	}

	// 设置字符串值
	err = s.store.MSet(ctx, map[string]interface{}{
		"test_str": "hello world",
	}, 0)
	if err != nil {
		s.t.Fatalf("MSet failed: %v", err)
	}

	// 再次获取
	found, err = s.store.Get(ctx, "test_str", &strVal)
	if err != nil {
		s.t.Fatalf("Get failed: %v", err)
	}
	if !found {
		s.t.Error("Expected found for existing key")
	}
	if strVal != "hello world" {
		s.t.Errorf("Expected 'hello world', got '%s'", strVal)
	}

	// 测试数字类型
	var intVal int
	err = s.store.MSet(ctx, map[string]interface{}{
		"test_int": 42,
	}, 0)
	if err != nil {
		s.t.Fatalf("MSet failed: %v", err)
	}

	found, err = s.store.Get(ctx, "test_int", &intVal)
	if err != nil {
		s.t.Fatalf("Get failed: %v", err)
	}
	if !found {
		s.t.Error("Expected found for existing key")
	}
	if intVal != 42 {
		s.t.Errorf("Expected 42, got %d", intVal)
	}

	// 测试结构体类型
	type Person struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	person := Person{Name: "Alice", Age: 30}
	err = s.store.MSet(ctx, map[string]interface{}{
		"test_struct": person,
	}, 0)
	if err != nil {
		s.t.Fatalf("MSet failed: %v", err)
	}

	var resultPerson Person
	found, err = s.store.Get(ctx, "test_struct", &resultPerson)
	if err != nil {
		s.t.Fatalf("Get failed: %v", err)
	}
	if !found {
		s.t.Error("Expected found for existing key")
	}
	if resultPerson.Name != "Alice" || resultPerson.Age != 30 {
		s.t.Errorf("Expected {Alice 30}, got %+v", resultPerson)
	}
}

// TestMGetMSet 测试批量操作
func (s *StoreTestSuite) TestMGetMSet() {
	ctx := context.Background()

	// 准备测试数据
	testData := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}

	// 批量设置
	err := s.store.MSet(ctx, testData, 0)
	if err != nil {
		s.t.Fatalf("MSet failed: %v", err)
	}

	// 批量获取
	keys := []string{"key1", "key2", "key3", "nonexistent"}
	var result map[string]interface{}
	err = s.store.MGet(ctx, keys, &result)
	if err != nil {
		s.t.Fatalf("MGet failed: %v", err)
	}

	if len(result) != 3 {
		s.t.Errorf("Expected 3 results, got %d", len(result))
	}

	// 验证结果
	if result["key1"] != "value1" {
		s.t.Errorf("Expected value1, got %v", result["key1"])
	}
	if result["key2"] != float64(123) { // JSON反序列化数字为float64
		s.t.Errorf("Expected 123, got %v", result["key2"])
	}
	if result["key3"] != true {
		s.t.Errorf("Expected true, got %v", result["key3"])
	}
}

// TestExists 测试键存在性检查
func (s *StoreTestSuite) TestExists() {
	ctx := context.Background()

	// 设置一些键
	err := s.store.MSet(ctx, map[string]interface{}{
		"exists1": "value1",
		"exists2": "value2",
	}, 0)
	if err != nil {
		s.t.Fatalf("MSet failed: %v", err)
	}

	// 检查存在性
	keys := []string{"exists1", "exists2", "nonexistent1", "nonexistent2"}
	existsMap, err := s.store.Exists(ctx, keys)
	if err != nil {
		s.t.Fatalf("Exists failed: %v", err)
	}

	expected := map[string]bool{
		"exists1":     true,
		"exists2":     true,
		"nonexistent1": false,
		"nonexistent2": false,
	}

	if !reflect.DeepEqual(existsMap, expected) {
		s.t.Errorf("Expected %v, got %v", expected, existsMap)
	}
}

// TestDel 测试删除操作
func (s *StoreTestSuite) TestDel() {
	ctx := context.Background()

	// 设置一些键
	err := s.store.MSet(ctx, map[string]interface{}{
		"del1": "value1",
		"del2": "value2",
		"del3": "value3",
	}, 0)
	if err != nil {
		s.t.Fatalf("MSet failed: %v", err)
	}

	// 删除键
	deleted, err := s.store.Del(ctx, "del1", "del2", "nonexistent")
	if err != nil {
		s.t.Fatalf("Del failed: %v", err)
	}
	if deleted != 2 {
		s.t.Errorf("Expected 2 deleted, got %d", deleted)
	}

	// 验证删除结果
	var val string
	found, err := s.store.Get(ctx, "del1", &val)
	if err != nil {
		s.t.Fatalf("Get failed: %v", err)
	}
	if found {
		s.t.Error("Expected not found for deleted key")
	}

	found, err = s.store.Get(ctx, "del3", &val)
	if err != nil {
		s.t.Fatalf("Get failed: %v", err)
	}
	if !found {
		s.t.Error("Expected found for non-deleted key")
	}
}

// TestTTL 测试TTL功能
func (s *StoreTestSuite) TestTTL() {
	ctx := context.Background()

	// 设置带TTL的键
	err := s.store.MSet(ctx, map[string]interface{}{
		"ttl_key": "ttl_value",
	}, 100*time.Millisecond)
	if err != nil {
		s.t.Fatalf("MSet failed: %v", err)
	}

	// 立即检查应该存在
	var val string
	found, err := s.store.Get(ctx, "ttl_key", &val)
	if err != nil {
		s.t.Fatalf("Get failed: %v", err)
	}
	if !found {
		s.t.Error("Expected found immediately after set")
	}

	// 等待过期
	time.Sleep(200 * time.Millisecond)

	// 检查应该不存在
	found, err = s.store.Get(ctx, "ttl_key", &val)
	if err != nil {
		s.t.Fatalf("Get failed: %v", err)
	}
	if found {
		s.t.Error("Expected not found after TTL expiration")
	}
}