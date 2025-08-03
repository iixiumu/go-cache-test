package ristretto

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRistrettoStore(t *testing.T) {
	store, err := NewRistrettoStore(1000)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("Get non-existent key", func(t *testing.T) {
		var result string
		found, err := store.Get(ctx, "nonexistent", &result)
		assert.NoError(t, err)
		assert.False(t, found)
	})

	t.Run("Set and Get", func(t *testing.T) {
		// 设置值
		err := store.MSet(ctx, map[string]interface{}{
			"key1": "value1",
			"key2": 123,
		}, time.Duration(0))
		require.NoError(t, err)

		// 获取字符串值
		var result1 string
		found, err := store.Get(ctx, "key1", &result1)
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "value1", result1)

		// 获取整数值
		var result2 interface{}
		found, err = store.Get(ctx, "key2", &result2)
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 123, result2)
	})

	t.Run("MGet", func(t *testing.T) {
		// 批量设置
		err := store.MSet(ctx, map[string]interface{}{
			"batch1": "value1",
			"batch2": "value2",
			"batch3": "value3",
		}, time.Duration(0))
		require.NoError(t, err)

		// 批量获取
		resultMap := make(map[string]interface{})
		err = store.MGet(ctx, []string{"batch1", "batch2", "batch3", "batch4"}, &resultMap)
		assert.NoError(t, err)
		assert.Equal(t, "value1", resultMap["batch1"])
		assert.Equal(t, "value2", resultMap["batch2"])
		assert.Equal(t, "value3", resultMap["batch3"])
		// batch4 不存在，应该没有这个键
		_, exists := resultMap["batch4"]
		assert.False(t, exists)
	})

	t.Run("Exists", func(t *testing.T) {
		// 设置一些键
		err := store.MSet(ctx, map[string]interface{}{
			"exist1": "value1",
			"exist2": "value2",
		}, time.Duration(0))
		require.NoError(t, err)

		// 检查存在性
		results, err := store.Exists(ctx, []string{"exist1", "exist2", "exist3"})
		assert.NoError(t, err)
		assert.True(t, results["exist1"])
		assert.True(t, results["exist2"])
		assert.False(t, results["exist3"])
	})

	t.Run("Del", func(t *testing.T) {
		// 设置一些键
		err := store.MSet(ctx, map[string]interface{}{
			"del1": "value1",
			"del2": "value2",
			"del3": "value3",
			"del4": "value4", // 添加del4用于测试
		}, 0)
		require.NoError(t, err)

		// 删除键
		count, err := store.Del(ctx, "del1", "del2", "del4")
		assert.NoError(t, err)
		assert.Equal(t, int64(3), count) // 修正期望值为3

		// 验证删除
		var result string
		found, err := store.Get(ctx, "del1", &result)
		assert.NoError(t, err)
		assert.False(t, found)

		found, err = store.Get(ctx, "del2", &result)
		assert.NoError(t, err)
		assert.False(t, found)

		// del3 应该还存在
		found, err = store.Get(ctx, "del3", &result)
		assert.NoError(t, err)
		assert.True(t, found)

		// del4 应该已被删除
		found, err = store.Get(ctx, "del4", &result)
		assert.NoError(t, err)
		assert.False(t, found)
	})
}