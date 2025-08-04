package cacher

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-cache/cacher/store/redis"
	"go-cache/cacher/store/ristretto"

	miniredis "github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

// TestCacherWithRistretto 测试使用Ristretto作为存储后端的Cacher
func TestCacherWithRistretto(t *testing.T) {
	// 创建Ristretto存储
	store, err := ristretto.NewRistrettoStore()
	if err != nil {
		t.Fatalf("Failed to create Ristretto store: %v", err)
	}

	// 创建Cacher
	cacher := NewCacher(store)

	// 运行所有测试
	testCacherGet(t, cacher)
	testCacherMGet(t, cacher)
	testCacherMDelete(t, cacher)
	testCacherMRefresh(t, cacher)
	testCacherWithFallback(t, cacher)
	testCacherWithBatchFallback(t, cacher)
}

// TestCacherWithRedis 测试使用Redis作为存储后端的Cacher
func TestCacherWithRedis(t *testing.T) {
	// 创建一个miniredis实例用于测试
	mr := miniredis.RunT(t)

	// 创建Redis客户端
	client := goredis.NewClient(&goredis.Options{
		Addr: mr.Addr(),
	})

	// 创建Redis存储
	store := redis.NewRedisStore(client)

	// 创建Cacher
	cacher := NewCacher(store)

	// 运行所有测试
	testCacherGet(t, cacher)
	testCacherMGet(t, cacher)
	testCacherMDelete(t, cacher)
	testCacherMRefresh(t, cacher)
	testCacherWithFallback(t, cacher)
	testCacherWithBatchFallback(t, cacher)
}

func testCacherGet(t *testing.T, cacher Cacher) {
	ctx := context.Background()

	// 测试带回退函数的Get
	t.Run("TestGetWithFallback", func(t *testing.T) {
		var result string
		fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
			if key == "test_key" {
				return "fallback_value", true, nil
			}
			return nil, false, nil
		}

		// 第一次调用应该触发回退函数
		found, err := cacher.Get(ctx, "test_key", &result, fallback, nil)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if !found {
			t.Fatalf("Expected key to be found")
		}
		if result != "fallback_value" {
			t.Fatalf("Expected fallback_value, got %s", result)
		}

		// 第二次调用应该从缓存获取
		result = ""
		found, err = cacher.Get(ctx, "test_key", &result, fallback, nil)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if !found {
			t.Fatalf("Expected key to be found")
		}
		if result != "fallback_value" {
			t.Fatalf("Expected fallback_value, got %s", result)
		}
	})

	// 测试带TTL的Get
	t.Run("TestGetWithTTL", func(t *testing.T) {
		var result string
		fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
			if key == "ttl_key" {
				return "ttl_value", true, nil
			}
			return nil, false, nil
		}

		// 设置带TTL的值
		opts := &CacheOptions{TTL: time.Millisecond * 100}
		found, err := cacher.Get(ctx, "ttl_key", &result, fallback, opts)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if !found {
			t.Fatalf("Expected key to be found")
		}
		if result != "ttl_value" {
			t.Fatalf("Expected ttl_value, got %s", result)
		}

		// 等待过期
		time.Sleep(time.Millisecond * 200)

		// 再次调用应该触发回退函数
		result = ""
		found, err = cacher.Get(ctx, "ttl_key", &result, fallback, opts)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if !found {
			t.Fatalf("Expected key to be found")
		}
		if result != "ttl_value" {
			t.Fatalf("Expected ttl_value, got %s", result)
		}
	})
}

func testCacherMGet(t *testing.T, cacher Cacher) {
	ctx := context.Background()

	// 测试带批量回退函数的MGet
	t.Run("TestMGetWithBatchFallback", func(t *testing.T) {
		batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
			result := make(map[string]interface{})
			for _, key := range keys {
				if key == "key1" {
					result[key] = "value1"
				} else if key == "key2" {
					result[key] = "value2"
				}
			}
			return result, nil
		}

		// 第一次调用应该触发批量回退函数
		keys := []string{"key1", "key2", "key3"}
		result := make(map[string]string)
		err := cacher.MGet(ctx, keys, &result, batchFallback, nil)
		if err != nil {
			t.Fatalf("MGet failed: %v", err)
		}

		// 验证结果
		if len(result) != 2 {
			t.Fatalf("Expected 2 results, got %d", len(result))
		}
		if result["key1"] != "value1" {
			t.Fatalf("Expected value1, got %s", result["key1"])
		}
		if result["key2"] != "value2" {
			t.Fatalf("Expected value2, got %s", result["key2"])
		}

		// 第二次调用应该从缓存获取
		result = make(map[string]string)
		err = cacher.MGet(ctx, keys, &result, batchFallback, nil)
		if err != nil {
			t.Fatalf("MGet failed: %v", err)
		}

		// 验证结果
		if len(result) != 2 {
			t.Fatalf("Expected 2 results, got %d", len(result))
		}
		if result["key1"] != "value1" {
			t.Fatalf("Expected value1, got %s", result["key1"])
		}
		if result["key2"] != "value2" {
			t.Fatalf("Expected value2, got %s", result["key2"])
		}
	})
}

func testCacherMDelete(t *testing.T, cacher Cacher) {
	ctx := context.Background()

	// 测试MDelete
	t.Run("TestMDelete", func(t *testing.T) {
		// 先设置一些值
		var result string
		fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
			if key == "delete_key1" {
				return "value1", true, nil
			} else if key == "delete_key2" {
				return "value2", true, nil
			}
			return nil, false, nil
		}

		// 确保值在缓存中
		cacher.Get(ctx, "delete_key1", &result, fallback, nil)
		cacher.Get(ctx, "delete_key2", &result, fallback, nil)

		// 删除键
		keys := []string{"delete_key1", "delete_key2", "nonexistent"}
		deleted, err := cacher.MDelete(ctx, keys)
		if err != nil {
			t.Fatalf("MDelete failed: %v", err)
		}
		if deleted != 2 {
			t.Fatalf("Expected 2 deleted keys, got %d", deleted)
		}

		// 验证键已被删除
		found, err := cacher.Get(ctx, "delete_key1", &result, func(ctx context.Context, key string) (interface{}, bool, error) {
			return nil, false, nil
		}, nil)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if found {
			t.Fatalf("Expected delete_key1 to be deleted")
		}

		found, err = cacher.Get(ctx, "delete_key2", &result, func(ctx context.Context, key string) (interface{}, bool, error) {
			return nil, false, nil
		}, nil)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if found {
			t.Fatalf("Expected delete_key2 to be deleted")
		}
	})
}

func testCacherMRefresh(t *testing.T, cacher Cacher) {
	ctx := context.Background()

	// 测试MRefresh
	t.Run("TestMRefresh", func(t *testing.T) {
		// 先设置一些值
		var result string
		fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
			if key == "refresh_key1" {
				return "old_value1", true, nil
			} else if key == "refresh_key2" {
				return "old_value2", true, nil
			}
			return nil, false, nil
		}

		// 确保值在缓存中
		cacher.Get(ctx, "refresh_key1", &result, fallback, nil)
		cacher.Get(ctx, "refresh_key2", &result, fallback, nil)

		// 刷新键
		batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
			result := make(map[string]interface{})
			for _, key := range keys {
				if key == "refresh_key1" {
					result[key] = "new_value1"
				} else if key == "refresh_key2" {
					result[key] = "new_value2"
				}
			}
			return result, nil
		}

		keys := []string{"refresh_key1", "refresh_key2"}
		resultMap := make(map[string]string)
		err := cacher.MRefresh(ctx, keys, &resultMap, batchFallback, nil)
		if err != nil {
			t.Fatalf("MRefresh failed: %v", err)
		}

		// 验证结果
		if len(resultMap) != 2 {
			t.Fatalf("Expected 2 results, got %d", len(resultMap))
		}
		if resultMap["refresh_key1"] != "new_value1" {
			t.Fatalf("Expected new_value1, got %s", resultMap["refresh_key1"])
		}
		if resultMap["refresh_key2"] != "new_value2" {
			t.Fatalf("Expected new_value2, got %s", resultMap["refresh_key2"])
		}

		// 验证缓存中的值已更新
		var value1 string
		cacher.Get(ctx, "refresh_key1", &value1, func(ctx context.Context, key string) (interface{}, bool, error) {
			return nil, false, nil
		}, nil)
		if value1 != "new_value1" {
			t.Fatalf("Expected new_value1 in cache, got %s", value1)
		}

		var value2 string
		cacher.Get(ctx, "refresh_key2", &value2, func(ctx context.Context, key string) (interface{}, bool, error) {
			return nil, false, nil
		}, nil)
		if value2 != "new_value2" {
			t.Fatalf("Expected new_value2 in cache, got %s", value2)
		}
	})
}

func testCacherWithFallback(t *testing.T, cacher Cacher) {
	ctx := context.Background()

	// 测试带错误的回退函数
	t.Run("TestGetWithFallbackError", func(t *testing.T) {
		var result string
		fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
			return nil, false, errors.New("fallback error")
		}

		// 调用应该返回错误
		found, err := cacher.Get(ctx, "error_key", &result, fallback, nil)
		if err == nil {
			t.Fatalf("Expected error, got nil")
		}
		if found {
			t.Fatalf("Expected key not to be found")
		}
	})
}

func testCacherWithBatchFallback(t *testing.T, cacher Cacher) {
	ctx := context.Background()

	// 测试带错误的批量回退函数
	t.Run("TestMGetWithBatchFallbackError", func(t *testing.T) {
		batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
			return nil, errors.New("batch fallback error")
		}

		// 调用应该返回错误
		keys := []string{"error_key1", "error_key2"}
		result := make(map[string]string)
		err := cacher.MGet(ctx, keys, &result, batchFallback, nil)
		if err == nil {
			t.Fatalf("Expected error, got nil")
		}
	})
}
