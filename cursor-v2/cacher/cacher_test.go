package cacher

import (
	"context"
	"testing"
	"time"

	redisStore "go-cache/cacher/store/redis"
	ristrettoStore "go-cache/cacher/store/ristretto"

	"github.com/alicebob/miniredis/v2"
	"github.com/dgraph-io/ristretto/v2"
	"github.com/redis/go-redis/v9"
)

// TestCacherWithRedis 测试基于Redis的Cacher
func TestCacherWithRedis(t *testing.T) {
	// 启动miniredis服务器
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	// 创建Redis Store
	store := redisStore.NewRedisStore(client)

	// 创建Cacher
	cacher := NewCacher(store)

	// 运行Cacher测试
	runCacherTests(t, cacher)
}

// TestCacherWithRistretto 测试基于Ristretto的Cacher
func TestCacherWithRistretto(t *testing.T) {
	// 创建Ristretto Store
	cache, err := ristretto.NewCache(&ristretto.Config[string, interface{}]{
		NumCounters: 1e7,     // 键跟踪数量
		MaxCost:     1 << 30, // 最大缓存大小(1GB)
		BufferItems: 64,      // 缓冲区大小
	})
	store, err := ristrettoStore.NewRistrettoStore(cache)
	if err != nil {
		t.Fatalf("Failed to create Ristretto store: %v", err)
	}

	// 创建Cacher
	cacher := NewCacher(store)

	// 运行Cacher测试
	runCacherTests(t, cacher)
}

// runCacherTests 运行Cacher的测试用例
func runCacherTests(t *testing.T, cacher Cacher) {
	t.Run("TestGetWithFallback", func(t *testing.T) {
		testGetWithFallback(t, cacher)
	})

	t.Run("TestMGetWithFallback", func(t *testing.T) {
		testMGetWithFallback(t, cacher)
	})

	t.Run("TestMDelete", func(t *testing.T) {
		testMDelete(t, cacher)
	})

	t.Run("TestMRefresh", func(t *testing.T) {
		testMRefresh(t, cacher)
	})

	t.Run("TestCacheOptions", func(t *testing.T) {
		testCacheOptions(t, cacher)
	})
}

// testGetWithFallback 测试带回退的Get操作
func testGetWithFallback(t *testing.T, cacher Cacher) {
	ctx := context.Background()

	// 模拟数据源
	dataSource := map[string]string{
		"user:1": "Alice",
		"user:2": "Bob",
		"user:3": "Charlie",
	}

	// 创建回退函数
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		if value, exists := dataSource[key]; exists {
			return value, true, nil
		}
		return nil, false, nil
	}

	// 测试获取存在的用户
	var result string
	found, err := cacher.Get(ctx, "user:1", &result, fallback, nil)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Fatal("Expected to find user:1")
	}
	if result != "Alice" {
		t.Fatalf("Expected 'Alice', got %q", result)
	}

	// 再次获取应该从缓存中获取
	var result2 string
	found, err = cacher.Get(ctx, "user:1", &result2, fallback, nil)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Fatal("Expected to find user:1 from cache")
	}
	if result2 != "Alice" {
		t.Fatalf("Expected 'Alice', got %q", result2)
	}

	// 测试获取不存在的用户
	var result3 string
	found, err = cacher.Get(ctx, "user:999", &result3, fallback, nil)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if found {
		t.Fatal("Expected user:999 to not exist")
	}
}

// testMGetWithFallback 测试带回退的MGet操作
func testMGetWithFallback(t *testing.T, cacher Cacher) {
	ctx := context.Background()

	// 模拟数据源
	dataSource := map[string]interface{}{
		"user:1": "Alice",
		"user:2": "Bob",
		"user:3": "Charlie",
		"user:4": "David",
	}

	// 创建批量回退函数
	batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		result := make(map[string]interface{})
		for _, key := range keys {
			if value, exists := dataSource[key]; exists {
				result[key] = value
			}
		}
		return result, nil
	}

	// 测试批量获取
	var resultMap map[string]interface{}
	err := cacher.MGet(ctx, []string{"user:1", "user:2", "user:3", "user:999"}, &resultMap, batchFallback, nil)
	if err != nil {
		t.Fatalf("MGet failed: %v", err)
	}

	// 验证结果
	if len(resultMap) != 3 {
		t.Fatalf("Expected 3 items, got %d", len(resultMap))
	}

	expected := map[string]interface{}{
		"user:1": "Alice",
		"user:2": "Bob",
		"user:3": "Charlie",
	}

	for key, expectedValue := range expected {
		if resultMap[key] != expectedValue {
			t.Fatalf("Expected %v for key %s, got %v", expectedValue, key, resultMap[key])
		}
	}
}

// testMDelete 测试批量删除
func testMDelete(t *testing.T, cacher Cacher) {
	ctx := context.Background()

	// 先设置一些数据
	dataSource := map[string]string{
		"delete:1": "value1",
		"delete:2": "value2",
		"delete:3": "value3",
	}

	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		if value, exists := dataSource[key]; exists {
			return value, true, nil
		}
		return nil, false, nil
	}

	// 获取数据以填充缓存
	var result string
	for key := range dataSource {
		cacher.Get(ctx, key, &result, fallback, nil)
	}

	// 删除部分键
	deleted, err := cacher.MDelete(ctx, []string{"delete:1", "delete:2"})
	if err != nil {
		t.Fatalf("MDelete failed: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("Expected 2 deleted, got %d", deleted)
	}

	// 验证删除结果
	var result2 string
	found, err := cacher.Get(ctx, "delete:1", &result2, fallback, nil)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	// 删除后再次获取会触发回退函数，所以应该能找到
	if !found {
		t.Fatal("Expected delete:1 to be found from fallback")
	}

	found, err = cacher.Get(ctx, "delete:3", &result2, fallback, nil)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Fatal("Expected delete:3 to still exist")
	}
}

// testMRefresh 测试批量刷新
func testMRefresh(t *testing.T, cacher Cacher) {
	ctx := context.Background()

	// 模拟数据源，每次调用返回不同的值
	dataSource := map[string]string{
		"refresh:1": "initial",
		"refresh:2": "initial",
	}

	batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		result := make(map[string]interface{})
		for _, key := range keys {
			if value, exists := dataSource[key]; exists {
				result[key] = value
			}
		}
		return result, nil
	}

	// 先获取数据
	var resultMap map[string]interface{}
	err := cacher.MGet(ctx, []string{"refresh:1", "refresh:2"}, &resultMap, batchFallback, nil)
	if err != nil {
		t.Fatalf("MGet failed: %v", err)
	}

	// 修改数据源
	dataSource["refresh:1"] = "updated"
	dataSource["refresh:2"] = "updated"

	// 刷新缓存
	err = cacher.MRefresh(ctx, []string{"refresh:1", "refresh:2"}, &resultMap, batchFallback, nil)
	if err != nil {
		t.Fatalf("MRefresh failed: %v", err)
	}

	// 验证刷新结果
	if resultMap["refresh:1"] != "updated" {
		t.Fatalf("Expected 'updated' for refresh:1, got %v", resultMap["refresh:1"])
	}
	if resultMap["refresh:2"] != "updated" {
		t.Fatalf("Expected 'updated' for refresh:2, got %v", resultMap["refresh:2"])
	}
}

// testCacheOptions 测试缓存选项
func testCacheOptions(t *testing.T, cacher Cacher) {
	ctx := context.Background()

	dataSource := map[string]string{
		"ttl:test": "value",
	}

	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		if value, exists := dataSource[key]; exists {
			return value, true, nil
		}
		return nil, false, nil
	}

	// 使用TTL选项
	opts := &CacheOptions{
		TTL: 100 * time.Millisecond,
	}

	var result string
	found, err := cacher.Get(ctx, "ttl:test", &result, fallback, opts)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Fatal("Expected to find ttl:test")
	}

	// 等待TTL过期
	time.Sleep(150 * time.Millisecond)

	// 再次获取应该触发回退
	found, err = cacher.Get(ctx, "ttl:test", &result, fallback, opts)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Fatal("Expected to find ttl:test from fallback")
	}
}
