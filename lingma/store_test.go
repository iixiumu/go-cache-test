package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMemoryStore_GetSet(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// 测试设置和获取
	items := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}

	err := store.MSet(ctx, items, 0)
	assert.NoError(t, err)

	// 测试获取字符串
	var strValue string
	found, err := store.Get(ctx, "key1", &strValue)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "value1", strValue)

	// 测试获取整数
	var intValue int
	found, err = store.Get(ctx, "key2", &intValue)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, 123, intValue)

	// 测试获取布尔值
	var boolValue bool
	found, err = store.Get(ctx, "key3", &boolValue)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, true, boolValue)

	// 测试获取不存在的键
	var missingValue string
	found, err = store.Get(ctx, "missing", &missingValue)
	assert.NoError(t, err)
	assert.False(t, found)
}

func TestMemoryStore_MGet(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// 设置测试数据
	items := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}

	err := store.MSet(ctx, items, 0)
	assert.NoError(t, err)

	// 测试批量获取
	keys := []string{"key1", "key2", "key3", "missing"}
	result := make(map[string]interface{})

	err = store.MGet(ctx, keys, &result)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(result))
	assert.Equal(t, "value1", result["key1"])
	assert.Equal(t, 123.0, result["key2"]) // JSON反序列化数字默认为float64
	assert.Equal(t, true, result["key3"])
}

func TestMemoryStore_Exists(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// 设置测试数据
	items := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	}

	err := store.MSet(ctx, items, 0)
	assert.NoError(t, err)

	// 测试存在性检查
	keys := []string{"key1", "key2", "missing"}
	result, err := store.Exists(ctx, keys)
	assert.NoError(t, err)
	assert.True(t, result["key1"])
	assert.True(t, result["key2"])
	assert.False(t, result["missing"])
}

func TestMemoryStore_TTL(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// 设置带TTL的项
	items := map[string]interface{}{
		"key1": "value1",
	}

	err := store.MSet(ctx, items, time.Millisecond*100)
	assert.NoError(t, err)

	// 立即获取应该成功
	var value string
	found, err := store.Get(ctx, "key1", &value)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "value1", value)

	// 等待过期
	time.Sleep(time.Millisecond * 150)

	// 再次获取应该失败
	found, err = store.Get(ctx, "key1", &value)
	assert.NoError(t, err)
	assert.False(t, found)
}

func TestMemoryStore_Del(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// 设置测试数据
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	err := store.MSet(ctx, items, 0)
	assert.NoError(t, err)

	// 删除部分键
	count, err := store.Del(ctx, "key1", "key2", "missing")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// 检查剩余的键
	var value string
	found, err := store.Get(ctx, "key1", &value)
	assert.NoError(t, err)
	assert.False(t, found)

	found, err = store.Get(ctx, "key2", &value)
	assert.NoError(t, err)
	assert.False(t, found)

	found, err = store.Get(ctx, "key3", &value)
	assert.NoError(t, err)
	assert.True(t, found)
}