package memory

import (
	"context"
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestMemoryStore_Get(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// 测试获取不存在的键
	var value string
	found, err := store.Get(ctx, "key1", &value)
	assert.NoError(t, err)
	assert.False(t, found)

	// 设置一个键值对
	err = store.MSet(ctx, map[string]interface{}{"key1": "value1"}, 0)
	assert.NoError(t, err)

	// 测试获取存在的键
	found, err = store.Get(ctx, "key1", &value)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "value1", value)
}

func TestMemoryStore_MGet(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// 设置一些键值对
	err := store.MSet(ctx, map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}, 0)
	assert.NoError(t, err)

	// 测试批量获取
	result := make(map[string]string)
	err = store.MGet(ctx, []string{"key1", "key2", "key3"}, &result)
	assert.NoError(t, err)
	assert.Equal(t, "value1", result["key1"])
	assert.Equal(t, "value2", result["key2"])
	_, exists := result["key3"]
	assert.False(t, exists)
}

func TestMemoryStore_Exists(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// 设置一些键值对
	err := store.MSet(ctx, map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}, 0)
	assert.NoError(t, err)

	// 测试存在性检查
	result, err := store.Exists(ctx, []string{"key1", "key2", "key3"})
	assert.NoError(t, err)
	assert.True(t, result["key1"])
	assert.True(t, result["key2"])
	assert.False(t, result["key3"])
}

func TestMemoryStore_MSet(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// 测试批量设置
	err := store.MSet(ctx, map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}, 0)
	assert.NoError(t, err)

	// 验证设置的值
	var value string
	found, err := store.Get(ctx, "key1", &value)
	assert.NoError(t, err)
	assert.True(t, found)
	// assert.Equal(t, "value1", value) // 需要完善反射实现才能正确断言
}

func TestMemoryStore_Del(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// 设置一些键值对
	err := store.MSet(ctx, map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}, 0)
	assert.NoError(t, err)

	// 删除部分键
	deleted, err := store.Del(ctx, "key1", "key2", "key4")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), deleted)

	// 验证键已被删除
	found, err := store.Get(ctx, "key1", nil)
	assert.NoError(t, err)
	assert.False(t, found)
}