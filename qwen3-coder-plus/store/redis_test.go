package store

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
)

func TestRedisStore(t *testing.T) {
	// 启动miniredis服务器
	s, err := miniredis.Run()
	assert.NoError(t, err)
	defer s.Close()

	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	// 创建Redis存储
	store := NewRedisStore(client)

	// 创建测试上下文
	ctx := context.Background()

	// 测试Set和Get
	t.Run("SetAndGet", func(t *testing.T) {
		// 设置键值对
		items := map[string]interface{}{
			"key1": "value1",
			"key2": 123,
		}
		err := store.MSet(ctx, items, time.Minute)
		assert.NoError(t, err)

		// 获取单个值
		var value string
		found, err := store.Get(ctx, "key1", &value)
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "value1", value)

		// 获取不存在的键
		found, err = store.Get(ctx, "nonexistent", &value)
		assert.NoError(t, err)
		assert.False(t, found)
	})

	// 测试MGet
	t.Run("MGet", func(t *testing.T) {
		// 设置测试数据
		items := map[string]interface{}{
			"mget1": "value1",
			"mget2": "value2",
		}
		err := store.MSet(ctx, items, time.Minute)
		assert.NoError(t, err)

		// 批量获取
		keys := []string{"mget1", "mget2", "nonexistent"}
		result := make(map[string]string)
		err = store.MGet(ctx, keys, &result)
		assert.NoError(t, err)

		// 验证结果
		assert.Equal(t, "value1", result["mget1"])
		assert.Equal(t, "value2", result["mget2"])
		_, exists := result["nonexistent"]
		assert.False(t, exists)
	})

	// 测试Exists
	t.Run("Exists", func(t *testing.T) {
		// 设置测试数据
		items := map[string]interface{}{
			"exist1": "value1",
		}
		err := store.MSet(ctx, items, time.Minute)
		assert.NoError(t, err)

		// 检查键存在性
		keys := []string{"exist1", "nonexistent"}
		result, err := store.Exists(ctx, keys)
		assert.NoError(t, err)
		assert.True(t, result["exist1"])
		assert.False(t, result["nonexistent"])
	})

	// 测试Del
	t.Run("Del", func(t *testing.T) {
		// 设置测试数据
		items := map[string]interface{}{
			"del1": "value1",
			"del2": "value2",
		}
		err := store.MSet(ctx, items, time.Minute)
		assert.NoError(t, err)

		// 删除键
		deleted, err := store.Del(ctx, "del1", "nonexistent")
		assert.NoError(t, err)
		assert.Equal(t, int64(1), deleted)

		// 验证键已被删除
		var value string
		found, err := store.Get(ctx, "del1", &value)
		assert.NoError(t, err)
		assert.False(t, found)
	})
}