package ristretto

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRistrettoStore_BasicOperations(t *testing.T) {
	// 创建Ristretto Store
	store, err := NewDefaultRistrettoStore()
	require.NoError(t, err)
	defer store.Close()

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
}
