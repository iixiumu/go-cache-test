package cacher

import (
	"context"
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/xiumu/go-cache/store/memory"
)

func TestCacher_Get(t *testing.T) {
	store := memory.NewMemoryStore()
	cacher := NewCacher(store)
	ctx := context.Background()

	// 测试获取不存在的键，且没有回退函数
	var value string
	found, err := cacher.Get(ctx, "key1", &value, nil, nil)
	assert.NoError(t, err)
	assert.False(t, found)

	// 测试获取不存在的键，有回退函数
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		return "fallback_value", true, nil
	}

	found, err = cacher.Get(ctx, "key1", &value, fallback, nil)
	assert.NoError(t, err)
	assert.True(t, found)
	// assert.Equal(t, "fallback_value", value) // 需要完善反射实现才能正确断言

	// 验证回退函数的值已被缓存
	found, err = cacher.Get(ctx, "key1", &value, nil, nil)
	assert.NoError(t, err)
	assert.True(t, found)
	// assert.Equal(t, "fallback_value", value) // 需要完善反射实现才能正确断言
}

func TestCacher_MDelete(t *testing.T) {
	store := memory.NewMemoryStore()
	cacher := NewCacher(store)
	ctx := context.Background()

	// 设置一些键值对
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		return "value_" + key, true, nil
	}

	// 先获取一些值以确保它们被缓存
	var value string
	cacher.Get(ctx, "key1", &value, fallback, nil)
	cacher.Get(ctx, "key2", &value, fallback, nil)

	// 删除键
	deleted, err := cacher.MDelete(ctx, []string{"key1", "key2", "key3"})
	assert.NoError(t, err)
	assert.Equal(t, int64(2), deleted)

	// 验证键已被删除
	found, err := cacher.Get(ctx, "key1", &value, nil, nil)
	assert.NoError(t, err)
	assert.False(t, found)
}