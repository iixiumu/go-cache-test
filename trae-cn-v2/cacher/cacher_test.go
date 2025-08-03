package cacher

import (
	"context"
	"testing"
	"time"

	"go-cache/cacher/store/ristretto"
)

// TestCacher 测试Cacher接口的实现
func TestCacher(t *testing.T) {
	// 创建Ristretto存储后端作为测试用例
	storeImpl, err := ristretto.NewRistrettoStore(1000)
	if err != nil {
		t.Fatalf("创建存储后端失败: %v", err)
	}

	// 创建Cacher实例
	cacher := NewCacher(storeImpl)

	ctx := context.Background()

	// 测试Get方法
	t.Run("Get", func(t *testing.T) {
		// 测试缓存命中
		key := "test_key"
		value := "test_value"

		// 先设置缓存
		if err := storeImpl.MSet(ctx, map[string]interface{}{key: value}, 0); err != nil {
			t.Fatalf("设置缓存失败: %v", err)
		}

		// 获取缓存
		var got string
		found, err := cacher.Get(ctx, key, &got, nil, nil)
		if err != nil {
			t.Fatalf("Get失败: %v", err)
		}
		if !found {
			t.Fatal("应该找到缓存项")
		}
		if got != value {
			t.Errorf("值不匹配: 期望 %s, 实际 %s", value, got)
		}

		// 测试缓存未命中，使用回退函数
		missingKey := "missing_key"
		fallbackValue := "fallback_value"

		var gotFallback string
		found, err = cacher.Get(ctx, missingKey, &gotFallback, func(ctx context.Context, key string) (interface{}, bool, error) {
			return fallbackValue, true, nil
		}, nil)
		if err != nil {
			t.Fatalf("Get失败: %v", err)
		}
		if !found {
			t.Fatal("应该找到缓存项")
		}
		if gotFallback != fallbackValue {
			t.Errorf("值不匹配: 期望 %s, 实际 %s", fallbackValue, gotFallback)
		}

		// 验证回退结果被缓存
		var cachedValue string
		found, err = storeImpl.Get(ctx, missingKey, &cachedValue)
		if err != nil {
			t.Fatalf("Get失败: %v", err)
		}
		if !found {
			t.Fatal("回退结果应该被缓存")
		}
		if cachedValue != fallbackValue {
			t.Errorf("缓存值不匹配: 期望 %s, 实际 %s", fallbackValue, cachedValue)
		}
	})

	// 测试MGet方法
	t.Run("MGet", func(t *testing.T) {
		// 设置一些初始缓存
		initialData := map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		}
		if err := storeImpl.MSet(ctx, initialData, 0); err != nil {
			t.Fatalf("设置初始缓存失败: %v", err)
		}

		// 批量获取，包括存在和不存在的键
		keys := []string{"key1", "key2", "key3", "key4"}
		var result map[string]interface{}

		// 定义回退函数
		fallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
			return map[string]interface{}{
				"key3": "fallback3",
				"key4": 100,
			}, nil
		}

		// 调用MGet
		if err := cacher.MGet(ctx, keys, &result, fallback, nil); err != nil {
			t.Fatalf("MGet失败: %v", err)
		}

		// 验证结果
		if result["key1"] != "value1" {
			t.Errorf("key1 值不匹配: 期望 value1, 实际 %v", result["key1"])
		}
		if result["key2"] != 42 {
			t.Errorf("key2 值不匹配: 期望 42, 实际 %v", result["key2"])
		}
		if result["key3"] != "fallback3" {
			t.Errorf("key3 值不匹配: 期望 fallback3, 实际 %v", result["key3"])
		}
		if result["key4"] != 100 {
			t.Errorf("key4 值不匹配: 期望 100, 实际 %v", result["key4"])
		}

		// 验证回退结果被缓存
		var cached3 string
		found, err := storeImpl.Get(ctx, "key3", &cached3)
		if err != nil {
			t.Fatalf("Get失败: %v", err)
		}
		if !found {
			t.Fatal("key3 应该被缓存")
		}
		if cached3 != "fallback3" {
			t.Errorf("key3 缓存值不匹配: 期望 fallback3, 实际 %s", cached3)
		}

		var cached4 int
		found, err = storeImpl.Get(ctx, "key4", &cached4)
		if err != nil {
			t.Fatalf("Get失败: %v", err)
		}
		if !found {
			t.Fatal("key4 应该被缓存")
		}
		if cached4 != 100 {
			t.Errorf("key4 缓存值不匹配: 期望 100, 实际 %d", cached4)
		}
	})

	// 测试MDelete方法
	t.Run("MDelete", func(t *testing.T) {
		// 设置一些缓存
		data := map[string]interface{}{
			"delete1": "value1",
			"delete2": "value2",
		}
		if err := storeImpl.MSet(ctx, data, 0); err != nil {
			t.Fatalf("设置缓存失败: %v", err)
		}

		// 删除缓存
		deleted, err := cacher.MDelete(ctx, []string{"delete1", "delete2", "nonexistent"})
		if err != nil {
			t.Fatalf("MDelete失败: %v", err)
		}
		if deleted != 2 {
			t.Errorf("删除数量不匹配: 期望 2, 实际 %d", deleted)
		}

		// 验证删除
		found, err := storeImpl.Get(ctx, "delete1", new(interface{}))
		if err != nil {
			t.Fatalf("Get失败: %v", err)
		}
		if found {
			t.Fatal("delete1 应该已删除")
		}
	})

	// 测试MRefresh方法
	t.Run("MRefresh", func(t *testing.T) {
		// 设置初始缓存
		key := "refresh_key"
		initialValue := "old_value"
		if err := storeImpl.MSet(ctx, map[string]interface{}{key: initialValue}, 0); err != nil {
			t.Fatalf("设置初始缓存失败: %v", err)
		}

		// 定义回退函数，返回新值
		fallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
			return map[string]interface{}{key: "new_value"}, nil
		}

		// 刷新缓存
		var result map[string]interface{}
		if err := cacher.MRefresh(ctx, []string{key}, &result, fallback, nil); err != nil {
			t.Fatalf("MRefresh失败: %v", err)
		}

		// 验证结果
		if result[key] != "new_value" {
			t.Errorf("值不匹配: 期望 new_value, 实际 %v", result[key])
		}

		// 验证缓存已更新
		var cachedValue string
		found, err := storeImpl.Get(ctx, key, &cachedValue)
		if err != nil {
			t.Fatalf("Get失败: %v", err)
		}
		if !found {
			t.Fatal("缓存应该存在")
		}
		if cachedValue != "new_value" {
			t.Errorf("缓存值不匹配: 期望 new_value, 实际 %s", cachedValue)
		}
	})

	// 测试TTL
	t.Run("TTL", func(t *testing.T) {
		key := "ttl_key"
		value := "ttl_value"

		// 使用回退函数设置带TTL的缓存
		var got string
		found, err := cacher.Get(ctx, key, &got, func(ctx context.Context, key string) (interface{}, bool, error) {
			return value, true, nil
		}, &CacheOptions{TTL: 1 * time.Second})
		if err != nil {
			t.Fatalf("Get失败: %v", err)
		}
		if !found {
			t.Fatal("应该找到缓存项")
		}

		// 立即检查缓存存在
		found, err = storeImpl.Get(ctx, key, new(interface{}))
		if err != nil {
			t.Fatalf("Get失败: %v", err)
		}
		if !found {
			t.Fatal("缓存应该存在")
		}

		// 等待过期
		time.Sleep(1100 * time.Millisecond)

		// 检查缓存是否过期
		found, err = storeImpl.Get(ctx, key, new(interface{}))
		if err != nil {
			t.Fatalf("Get失败: %v", err)
		}
		if found {
			t.Fatal("缓存应该已过期")
		}
	})
}