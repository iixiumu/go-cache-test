package cache

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/dgraph-io/ristretto"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacher(t *testing.T) {
	// 使用内存存储进行测试
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,
		MaxCost:     1 << 30,
		BufferItems: 64,
	})
	require.NoError(t, err)

	store := NewRistrettoStore(cache)
	cacher := NewCacher(store)

	// 测试基本功能
	t.Run("BasicGet", func(t *testing.T) {
		testCacherBasicGet(t, cacher)
	})

	t.Run("MGet", func(t *testing.T) {
		testCacherMGet(t, cacher, store)
	})

	t.Run("MDelete", func(t *testing.T) {
		testCacherMDelete(t, cacher, store)
	})

	t.Run("MRefresh", func(t *testing.T) {
		testCacherMRefresh(t, cacher, store)
	})

	t.Run("Fallback", func(t *testing.T) {
		testCacherFallback(t, cacher)
	})

	t.Run("BatchFallback", func(t *testing.T) {
		testCacherBatchFallback(t, cacher)
	})

	t.Run("CacheMiss", func(t *testing.T) {
		testCacherCacheMiss(t, cacher)
	})

	t.Run("WithTTL", func(t *testing.T) {
		testCacherWithTTL(t, cacher)
	})
}

func testCacherBasicGet(t *testing.T, cacher Cacher) {
	ctx := context.Background()

	// 测试缓存未命中且没有回退函数
	var value string
	found, err := cacher.Get(ctx, "nonexistent", &value, nil, nil)
	assert.NoError(t, err)
	assert.False(t, found)
	assert.Empty(t, value)

	// 测试带回退函数的缓存未命中
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		if key == "test" {
			return "fallback_value", true, nil
		}
		return nil, false, nil
	}

	found, err = cacher.Get(ctx, "test", &value, fallback, nil)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "fallback_value", value)

	// 验证值已被缓存
	found, err = cacher.Get(ctx, "test", &value, nil, nil)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "fallback_value", value)
}

func testCacherMGet(t *testing.T, cacher Cacher, store Store) {
	ctx := context.Background()

	// 预填充一些缓存
	items := map[string]interface{}{
		"key1": "value_key1",
		"key2": "value_key2",
	}
	err := store.MSet(ctx, items, 0)
	assert.NoError(t, err)

	// 测试MGet
	result := make(map[string]string)
	err = cacher.MGet(ctx, []string{"key1", "key2", "key3"}, &result, func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		fallbackData := make(map[string]interface{})
		for _, key := range keys {
			fallbackData[key] = "fallback_" + key
		}
		return fallbackData, nil
	}, nil)

	assert.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, "value_key1", result["key1"])
	assert.Equal(t, "value_key2", result["key2"])
	assert.Equal(t, "fallback_key3", result["key3"])
}

func testCacherMDelete(t *testing.T, cacher Cacher, store Store) {
	ctx := context.Background()

	// 预填充缓存
	items := map[string]interface{}{
		"del1": "value_del1",
		"del2": "value_del2",
		"del3": "value_del3",
	}
	err := store.MSet(ctx, items, 0)
	assert.NoError(t, err)

	// 验证存在
	var value string
	found, err := cacher.Get(ctx, "del1", &value, nil, nil)
	assert.NoError(t, err)
	assert.True(t, found)

	// 删除
	deleted, err := cacher.MDelete(ctx, []string{"del1", "del2"})
	assert.NoError(t, err)
	assert.Equal(t, int64(2), deleted)

	// 验证已删除
	found, err = cacher.Get(ctx, "del1", &value, nil, nil)
	assert.NoError(t, err)
	assert.False(t, found)

	var value2 string
	found, err = cacher.Get(ctx, "del3", &value2, nil, nil)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "value_del3", value2)
}

func testCacherMRefresh(t *testing.T, cacher Cacher, store Store) {
	ctx := context.Background()

	// 预填充缓存
	items := map[string]interface{}{
		"refresh1": "old_refresh1",
		"refresh2": "old_refresh2",
	}
	err := store.MSet(ctx, items, 0)
	assert.NoError(t, err)

	// 验证旧值
	var value string
	found, err := cacher.Get(ctx, "refresh1", &value, nil, nil)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "old_refresh1", value)

	// 刷新
	newFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		newData := make(map[string]interface{})
		for _, key := range keys {
			newData[key] = "new_" + key
		}
		return newData, nil
	}

	result := make(map[string]string)
	err = cacher.MRefresh(ctx, []string{"refresh1", "refresh2"}, &result, newFallback, nil)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "new_refresh1", result["refresh1"])
	assert.Equal(t, "new_refresh2", result["refresh2"])

	// 验证新值已缓存
	found, err = cacher.Get(ctx, "refresh1", &value, nil, nil)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "new_refresh1", value)
}

func testCacherFallback(t *testing.T, cacher Cacher) {
	ctx := context.Background()

	// 测试回退函数错误
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		return nil, false, errors.New("fallback error")
	}

	var value string
	found, err := cacher.Get(ctx, "error_key", &value, fallback, nil)
	assert.Error(t, err)
	assert.Equal(t, "fallback error", err.Error())
	assert.False(t, found)

	// 测试回退函数返回未找到
	fallbackNotFound := func(ctx context.Context, key string) (interface{}, bool, error) {
		return nil, false, nil
	}

	found, err = cacher.Get(ctx, "not_found", &value, fallbackNotFound, nil)
	assert.NoError(t, err)
	assert.False(t, found)
}

func testCacherBatchFallback(t *testing.T, cacher Cacher) {
	ctx := context.Background()

	// 测试批量回退函数错误
	batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		return nil, errors.New("batch fallback error")
	}

	result := make(map[string]string)
	err := cacher.MGet(ctx, []string{"key1", "key2"}, &result, batchFallback, nil)
	assert.Error(t, err)
	assert.Equal(t, "batch fallback error", err.Error())

	// 测试部分回退
	partialFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		partialResult := make(map[string]interface{})
		// 只返回部分键
		for _, key := range keys {
			if key == "partial1" {
				partialResult[key] = "partial_value"
			}
		}
		return partialResult, nil
	}

	result2 := make(map[string]string) // 使用新变量名避免冲突
	err = cacher.MGet(ctx, []string{"partial1", "partial2"}, &result2, partialFallback, nil)
	assert.NoError(t, err)
	assert.Len(t, result2, 1)
	assert.Equal(t, "partial_value", result2["partial1"])
}

func testCacherCacheMiss(t *testing.T, cacher Cacher) {
	ctx := context.Background()

	// 测试空键列表
	result := make(map[string]string)
	err := cacher.MGet(ctx, []string{}, &result, nil, nil)
	assert.NoError(t, err)
	assert.Empty(t, result)

	// 测试nil回退函数
	err = cacher.MGet(ctx, []string{"nonexistent"}, &result, nil, nil)
	assert.NoError(t, err)
	assert.Empty(t, result)
}

func testCacherWithTTL(t *testing.T, cacher Cacher) {
	ctx := context.Background()

	// 测试带TTL的缓存
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		return "ttl_value", true, nil
	}

	var value string
	found, err := cacher.Get(ctx, "ttl_key", &value, fallback, &CacheOptions{TTL: 100 * time.Millisecond})
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "ttl_value", value)
}

func TestCacherWithLock(t *testing.T) {
	// 测试带锁的Cacher
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,
		MaxCost:     1 << 30,
		BufferItems: 64,
	})
	require.NoError(t, err)

	store := NewRistrettoStore(cache)
	cacher := NewCacherWithLock(store)

	ctx := context.Background()
	var counter int
	var mu sync.Mutex

	// 测试并发回退
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		mu.Lock()
		counter++
		mu.Unlock()
		
		// 模拟耗时操作
		time.Sleep(10 * time.Millisecond)
		return "concurrent_value", true, nil
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var value string
			found, err := cacher.Get(ctx, "concurrent_key", &value, fallback, nil)
			assert.NoError(t, err)
			assert.True(t, found)
			assert.Equal(t, "concurrent_value", value)
		}()
	}

	wg.Wait()
	
	// 验证回退函数只执行一次
	assert.Equal(t, 1, counter)
}

func TestCacherWithRedis(t *testing.T) {
	miniredis := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{
		Addr: miniredis.Addr(),
	})
	defer rdb.Close()

	store := NewRedisStore(rdb)
	cacher := NewCacher(store)

	ctx := context.Background()

	// 测试Redis存储的完整流程
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		return "redis_value", true, nil
	}

	var value string
	found, err := cacher.Get(ctx, "redis_key", &value, fallback, nil)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "redis_value", value)

	// 验证值已存储到Redis
	var redisValue string
	found, err = store.Get(ctx, "redis_key", &redisValue)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "redis_value", redisValue)
}