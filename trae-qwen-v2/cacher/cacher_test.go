package cacher

import (
	"context"
	"testing"
	"time"
)

// TestCacher 测试Cacher接口实现
type TestCacher struct {
	cacher Cacher
}

// NewTestCacher 创建新的Cacher测试器
func NewTestCacher(cacher Cacher) *TestCacher {
	return &TestCacher{cacher: cacher}
}

// TestGet 测试Get方法
func (tc *TestCacher) TestGet(t *testing.T) {
	ctx := context.Background()
	key := "test_key"
	expectedValue := "test_value"

	// 定义回退函数
	fallback := func(ctx context.Context, k string) (interface{}, bool, error) {
		if k == key {
			return expectedValue, true, nil
		}
		return nil, false, nil
	}

	// 获取值
	var result string
	found, err := tc.cacher.Get(ctx, key, &result, fallback, nil)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Fatalf("Get: key not found")
	}
	if result != expectedValue {
		t.Fatalf("Get: expected %v, got %v", expectedValue, result)
	}

	// 再次获取值，应该从缓存中获取
	var result2 string
	found2, err2 := tc.cacher.Get(ctx, key, &result2, fallback, nil)
	if err2 != nil {
		t.Fatalf("Get failed: %v", err2)
	}
	if !found2 {
		t.Fatalf("Get: key not found")
	}
	if result2 != expectedValue {
		t.Fatalf("Get: expected %v, got %v", expectedValue, result2)
	}
}

// TestMGet 测试MGet方法
func (tc *TestCacher) TestMGet(t *testing.T) {
	ctx := context.Background()
	keys := []string{"key1", "key2", "key3"}
	expectedValues := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	// 定义批量回退函数
	batchFallback := func(ctx context.Context, ks []string) (map[string]interface{}, error) {
		result := make(map[string]interface{})
		for _, k := range ks {
			if v, ok := expectedValues[k]; ok {
				result[k] = v
			}
		}
		return result, nil
	}

	// 批量获取值
	result := make(map[string]string)
	if err := tc.cacher.MGet(ctx, keys, &result, batchFallback, nil); err != nil {
		t.Fatalf("MGet failed: %v", err)
	}

	// 验证结果
	for k, v := range expectedValues {
		if result[k] != v {
			t.Fatalf("MGet: expected %v for key %s, got %v", v, k, result[k])
		}
	}
}

// TestMDelete 测试MDelete方法
func (tc *TestCacher) TestMDelete(t *testing.T) {
	ctx := context.Background()
	key := "test_key"
	expectedValue := "test_value"

	// 定义回退函数
	fallback := func(ctx context.Context, k string) (interface{}, bool, error) {
		if k == key {
			return expectedValue, true, nil
		}
		return nil, false, nil
	}

	// 获取值以确保它在缓存中
	var result string
	_, err := tc.cacher.Get(ctx, key, &result, fallback, nil)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// 删除值
	deleted, err := tc.cacher.MDelete(ctx, []string{key})
	if err != nil {
		t.Fatalf("MDelete failed: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("MDelete: expected 1 key deleted, got %d", deleted)
	}

	// 再次获取值，应该从回退函数获取
	var result2 string
	found, err := tc.cacher.Get(ctx, key, &result2, fallback, nil)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Fatalf("Get: key not found")
	}
	if result2 != expectedValue {
		t.Fatalf("Get: expected %v, got %v", expectedValue, result2)
	}
}

// TestMRefresh 测试MRefresh方法
func (tc *TestCacher) TestMRefresh(t *testing.T) {
	ctx := context.Background()
	key := "test_key"
	initialValue := "initial_value"
	refreshedValue := "refreshed_value"

	// 定义回退函数
	fallbackCount := 0
	fallback := func(ctx context.Context, k string) (interface{}, bool, error) {
		fallbackCount++
		if k == key {
			if fallbackCount == 1 {
				return initialValue, true, nil
			}
			return refreshedValue, true, nil
		}
		return nil, false, nil
	}

	// 定义批量回退函数
	batchFallback := func(ctx context.Context, ks []string) (map[string]interface{}, error) {
		result := make(map[string]interface{})
		for _, k := range ks {
			if k == key {
				result[k] = refreshedValue
			}
		}
		return result, nil
	}

	// 获取值
	var result string
	_, err := tc.cacher.Get(ctx, key, &result, fallback, nil)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if result != initialValue {
		t.Fatalf("Get: expected %v, got %v", initialValue, result)
	}

	// 刷新值
	refreshResult := make(map[string]string)
	if err := tc.cacher.MRefresh(ctx, []string{key}, &refreshResult, batchFallback, nil); err != nil {
		t.Fatalf("MRefresh failed: %v", err)
	}
	if refreshResult[key] != refreshedValue {
		t.Fatalf("MRefresh: expected %v, got %v", refreshedValue, refreshResult[key])
	}

	// 再次获取值，应该从缓存中获取刷新后的值
	var result2 string
	_, err = tc.cacher.Get(ctx, key, &result2, fallback, nil)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if result2 != refreshedValue {
		t.Fatalf("Get: expected %v, got %v", refreshedValue, result2)
	}
}

// TestCacheOptions 测试缓存选项
func (tc *TestCacher) TestCacheOptions(t *testing.T) {
	ctx := context.Background()
	key := "test_key"
	expectedValue := "test_value"

	// 定义回退函数
	fallback := func(ctx context.Context, k string) (interface{}, bool, error) {
		if k == key {
			return expectedValue, true, nil
		}
		return nil, false, nil
	}

	// 使用TTL选项
	opts := &CacheOptions{
		TTL: 1 * time.Second,
	}

	// 获取值
	var result string
	_, err := tc.cacher.Get(ctx, key, &result, fallback, opts)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if result != expectedValue {
		t.Fatalf("Get: expected %v, got %v", expectedValue, result)
	}

	// 等待TTL过期
	time.Sleep(1 * time.Second)

	// 再次获取值，应该从回退函数获取
	var result2 string
	_, err = tc.cacher.Get(ctx, key, &result2, fallback, opts)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if result2 != expectedValue {
		t.Fatalf("Get: expected %v, got %v", expectedValue, result2)
	}
}