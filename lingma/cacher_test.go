package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCacher_GetWithFallback(t *testing.T) {
	store := NewMemoryStore()
	cacher := NewCacher(store)
	ctx := context.Background()

	// 定义回退函数
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		if key == "test_key" {
			return "fallback_value", true, nil
		}
		return nil, false, nil
	}

	// 测试缓存未命中时使用回退函数
	var value string
	found, err := cacher.Get(ctx, "test_key", &value, fallback, nil)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "fallback_value", value)

	// 再次获取应该从缓存中获取
	value = ""
	found, err = cacher.Get(ctx, "test_key", &value, nil, nil) // 不提供回退函数
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "fallback_value", value)
}

func TestCacher_GetWithoutFallback(t *testing.T) {
	store := NewMemoryStore()
	cacher := NewCacher(store)
	ctx := context.Background()

	// 测试没有回退函数且缓存未命中
	var value string
	found, err := cacher.Get(ctx, "missing_key", &value, nil, nil)
	assert.NoError(t, err)
	assert.False(t, found)
}

func TestCacher_MGet(t *testing.T) {
	store := NewMemoryStore()
	cacher := NewCacher(store)
	ctx := context.Background()

	// 预先设置一些缓存
	items := map[string]interface{}{
		"cached_key1": "cached_value1",
		"cached_key2": "cached_value2",
	}
	err := store.MSet(ctx, items, 0)
	assert.NoError(t, err)

	// 定义批量回退函数
	batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		result := make(map[string]interface{})
		for _, key := range keys {
			if key == "fallback_key1" || key == "fallback_key2" {
				result[key] = "fallback_" + key
			}
		}
		return result, nil
	}

	// 测试批量获取
	keys := []string{"cached_key1", "cached_key2", "fallback_key1", "fallback_key2", "missing_key"}
	result := make(map[string]interface{})
	err = cacher.MGet(ctx, keys, &result, batchFallback, nil)
	assert.NoError(t, err)
	assert.Equal(t, 4, len(result))
	assert.Equal(t, "cached_value1", result["cached_key1"])
	assert.Equal(t, "cached_value2", result["cached_key2"])
	assert.Equal(t, "fallback_fallback_key1", result["fallback_key1"])
	assert.Equal(t, "fallback_fallback_key2", result["fallback_key2"])

	// 再次获取所有键，应该都从缓存中获取
	result = make(map[string]interface{})
	err = cacher.MGet(ctx, keys, &result, nil, nil) // 不提供回退函数
	assert.NoError(t, err)
	assert.Equal(t, 4, len(result)) // missing_key还是不存在
}

func TestCacher_MDelete(t *testing.T) {
	store := NewMemoryStore()
	cacher := NewCacher(store)
	ctx := context.Background()

	// 预先设置一些缓存
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	err := store.MSet(ctx, items, 0)
	assert.NoError(t, err)

	// 删除部分键
	count, err := cacher.MDelete(ctx, []string{"key1", "key2", "missing"})
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// 检查剩余的键
	var value string
	found, err := cacher.Get(ctx, "key1", &value, nil, nil)
	assert.NoError(t, err)
	assert.False(t, found)

	found, err = cacher.Get(ctx, "key2", &value, nil, nil)
	assert.NoError(t, err)
	assert.False(t, found)

	found, err = cacher.Get(ctx, "key3", &value, nil, nil)
	assert.NoError(t, err)
	assert.True(t, found)
}

func TestCacher_MRefresh(t *testing.T) {
	store := NewMemoryStore()
	cacher := NewCacher(store)
	ctx := context.Background()

	// 预先设置一些缓存
	items := map[string]interface{}{
		"key1": "old_value1",
		"key2": "old_value2",
	}
	err := store.MSet(ctx, items, 0)
	assert.NoError(t, err)

	// 定义批量回退函数用于刷新
	batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		result := make(map[string]interface{})
		for _, key := range keys {
			result[key] = "new_" + key
		}
		return result, nil
	}

	// 刷新缓存
	keys := []string{"key1", "key2", "key3"}
	result := make(map[string]interface{})
	err = cacher.MRefresh(ctx, keys, &result, batchFallback, nil)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(result))
	assert.Equal(t, "new_key1", result["key1"])
	assert.Equal(t, "new_key2", result["key2"])
	assert.Equal(t, "new_key3", result["key3"])

	// 检查缓存是否真的被更新
	var value string
	found, err := cacher.Get(ctx, "key1", &value, nil, nil)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "new_key1", value)
}

func TestCacher_GetWithTTL(t *testing.T) {
	store := NewMemoryStore()
	cacher := NewCacher(store)
	ctx := context.Background()

	// 定义回退函数
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		return "ttl_value", true, nil
	}

	// 使用TTL选项获取
	opts := &CacheOptions{
		TTL: time.Millisecond * 100,
	}
	var value string
	found, err := cacher.Get(ctx, "ttl_key", &value, fallback, opts)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "ttl_value", value)

	// 等待过期
	time.Sleep(time.Millisecond * 150)

	// 再次获取应该重新触发回退函数
	found, err = cacher.Get(ctx, "ttl_key", &value, fallback, opts)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "ttl_value", value)
}