package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisStore_BasicOperations(t *testing.T) {
	// 启动miniredis
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	// 创建Redis Store
	store := NewRedisStore(client)
	ctx := context.Background()

	t.Run("GetSet", func(t *testing.T) {
		// 测试不存在的键
		var result string
		found, err := store.Get(ctx, "nonexistent", &result)
		assert.NoError(t, err)
		assert.False(t, found)

		// 设置一个值
		items := map[string]interface{}{
			"test_key": "test_value",
		}
		err = store.MSet(ctx, items, 0)
		require.NoError(t, err)

		// 获取值
		found, err = store.Get(ctx, "test_key", &result)
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "test_value", result)
	})

	t.Run("MGetMSet", func(t *testing.T) {
		// 设置测试数据
		items := map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		}
		err := store.MSet(ctx, items, 0)
		require.NoError(t, err)

		// 批量获取
		keys := []string{"key1", "key2", "key3", "nonexistent"}
		result := make(map[string]string)
		err = store.MGet(ctx, keys, &result)
		assert.NoError(t, err)

		// 验证结果
		assert.Equal(t, "value1", result["key1"])
		assert.Equal(t, "value2", result["key2"])
		assert.Equal(t, "value3", result["key3"])
		assert.NotContains(t, result, "nonexistent")
	})

	t.Run("Exists", func(t *testing.T) {
		// 设置测试数据
		items := map[string]interface{}{
			"existing1": "value1",
			"existing2": "value2",
		}
		err := store.MSet(ctx, items, 0)
		require.NoError(t, err)

		// 检查存在性
		keys := []string{"existing1", "existing2", "nonexistent"}
		result, err := store.Exists(ctx, keys)
		assert.NoError(t, err)

		assert.True(t, result["existing1"])
		assert.True(t, result["existing2"])
		assert.False(t, result["nonexistent"])
	})

	t.Run("Del", func(t *testing.T) {
		// 设置测试数据
		items := map[string]interface{}{
			"del_key1": "value1",
			"del_key2": "value2",
		}
		err := store.MSet(ctx, items, 0)
		require.NoError(t, err)

		// 删除键
		deleted, err := store.Del(ctx, "del_key1", "nonexistent")
		assert.NoError(t, err)
		assert.Equal(t, int64(1), deleted)

		// 验证删除结果
		var result string
		found, err := store.Get(ctx, "del_key1", &result)
		assert.NoError(t, err)
		assert.False(t, found)
	})

	t.Run("TTL", func(t *testing.T) {
		// 设置带TTL的数据
		items := map[string]interface{}{
			"ttl_key": "ttl_value",
		}
		err := store.MSet(ctx, items, 1*time.Second)
		assert.NoError(t, err)

		// 立即检查应该存在
		var result string
		found, err := store.Get(ctx, "ttl_key", &result)
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "ttl_value", result)

		// 手动触发过期（miniredis需要手动触发）
		mr.FastForward(2 * time.Second)

		// 检查应该已过期
		found, err = store.Get(ctx, "ttl_key", &result)
		assert.NoError(t, err)
		assert.False(t, found)
	})
}
