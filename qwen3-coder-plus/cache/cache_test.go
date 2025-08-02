package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCacher(t *testing.T) {
	// 创建内存存储
	memStore := newTestMemoryStore()

	// 创建缓存器
	cacher := New(memStore)

	// 创建测试上下文
	ctx := context.Background()

	// 测试Get方法
	t.Run("Get", func(t *testing.T) {
		// 定义回退函数
		fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
			if key == "test_key" {
				return "fallback_value", true, nil
			}
			return nil, false, nil
		}

		// 从回退函数获取数据
		var value string
		found, err := cacher.Get(ctx, "test_key", &value, fallback, &CacheOptions{TTL: time.Minute})
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "fallback_value", value)

		// 再次获取同样的键，这次应该直接从缓存中获取
		found, err = cacher.Get(ctx, "test_key", &value, fallback, &CacheOptions{TTL: time.Minute})
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "fallback_value", value)
	})

	// 测试Get方法无回退函数
	t.Run("GetWithoutFallback", func(t *testing.T) {
		var value string
		found, err := cacher.Get(ctx, "nonexistent", &value, nil, &CacheOptions{TTL: time.Minute})
		assert.NoError(t, err)
		assert.False(t, found)
	})

	// 测试MGet方法
	t.Run("MGet", func(t *testing.T) {
		// 定义批量回退函数
		batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
			result := make(map[string]interface{})
			for _, key := range keys {
				if key == "mget1" || key == "mget2" {
					result[key] = key + "_value"
				}
			}
			return result, nil
		}

		// 批量获取数据
		keys := []string{"mget1", "mget2", "nonexistent"}
		result := make(map[string]string)
		err := cacher.MGet(ctx, keys, &result, batchFallback, &CacheOptions{TTL: time.Minute})
		assert.NoError(t, err)

		// 验证结果
		assert.Equal(t, "mget1_value", result["mget1"])
		assert.Equal(t, "mget2_value", result["mget2"])
		_, exists := result["nonexistent"]
		assert.False(t, exists)

		// 再次批量获取，应该直接从缓存中获取
		result2 := make(map[string]string)
		err = cacher.MGet(ctx, keys, &result2, batchFallback, &CacheOptions{TTL: time.Minute})
		assert.NoError(t, err)

		// 验证结果
		assert.Equal(t, "mget1_value", result2["mget1"])
		assert.Equal(t, "mget2_value", result2["mget2"])
		_, exists = result2["nonexistent"]
		assert.False(t, exists)
	})

	// 测试MDelete方法
	t.Run("MDelete", func(t *testing.T) {
		// 设置测试数据
		memStore.MSet(ctx, map[string]interface{}{
			"delete1": "value1",
			"delete2": "value2",
		}, time.Minute)

		// 删除键
		deleted, err := cacher.MDelete(ctx, []string{"delete1", "nonexistent"})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), deleted)

		// 验证键已被删除
		var value string
		found, err := cacher.Get(ctx, "delete1", &value, nil, &CacheOptions{TTL: time.Minute})
		assert.NoError(t, err)
		assert.False(t, found)
	})

	// 测试MRefresh方法
	t.Run("MRefresh", func(t *testing.T) {
		// 定义批量回退函数
		batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
			result := make(map[string]interface{})
			for _, key := range keys {
				result[key] = key + "_new_value"
			}
			return result, nil
		}

		// 刷新缓存项
		keys := []string{"refresh1", "refresh2"}
		result := make(map[string]string)
		err := cacher.MRefresh(ctx, keys, &result, batchFallback, &CacheOptions{TTL: time.Minute})
		assert.NoError(t, err)

		// 验证结果
		assert.Equal(t, "refresh1_new_value", result["refresh1"])
		assert.Equal(t, "refresh2_new_value", result["refresh2"])

		// 验证数据已存入缓存
		var value string
		found, err := cacher.Get(ctx, "refresh1", &value, nil, &CacheOptions{TTL: time.Minute})
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "refresh1_new_value", value)
	})
}