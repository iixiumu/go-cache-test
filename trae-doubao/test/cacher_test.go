package test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/xiumu/git/me/go-cache/trae/pkg/cacher"
	"github.com/xiumu/git/me/go-cache/trae/pkg/store"
	"github.com/xiumu/git/me/go-cache/trae/pkg/store/gcache"
	bluelegcache "github.com/bluele/gcache"
)

// TestCacher 测试Cacher接口的通用测试函数
func TestCacher(t *testing.T, c cacher.Cacher) {
	ctx := context.Background()

	// 测试Get方法
	// 定义回退函数
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		if key == "key1" {
			return "value1", true, nil
		}
		if key == "key2" {
			return 42, true, nil
		}
		if key == "key3" {
			return map[string]interface{}{"subkey": "subvalue"}, true, nil
		}
		return nil, false, nil
	}

	// 测试缓存未命中时调用回退函数
	var val1 string
	found, err := c.Get(ctx, "key1", &val1, fallback, nil)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "value1", val1)

	// 测试缓存命中时不调用回退函数
	var val1Again string
	found, err = c.Get(ctx, "key1", &val1Again, func(ctx context.Context, key string) (interface{}, bool, error) {
		// 这个函数不应该被调用
		return nil, false, errors.New("should not be called")
	}, nil)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "value1", val1Again)

	// 测试MGet方法
	// 定义批量回退函数
	batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		result := make(map[string]interface{})
		for _, key := range keys {
			if key == "key4" {
				result[key] = "value4"
			}
			if key == "key5" {
				result[key] = 100
			}
		}
		return result, nil
	}

	// 先缓存key2
	var val2 int
	found, err = c.Get(ctx, "key2", &val2, fallback, nil)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, 42, val2)

	// 批量获取，部分命中
	results := make(map[string]interface{})
	assert.NoError(t, c.MGet(ctx, []string{"key2", "key4", "key6"}, &results, batchFallback, nil))
	assert.Equal(t, 42, results["key2"])
	assert.Equal(t, "value4", results["key4"])
	assert.Nil(t, results["key6"])

	// 测试MDelete方法
	count, err := c.MDelete(ctx, []string{"key1", "key2"})
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// 检查是否已删除
	var deletedVal string
	found, err = c.Get(ctx, "key1", &deletedVal, nil, nil)
	assert.NoError(t, err)
	assert.False(t, found)

	// 测试MRefresh方法
	// 先缓存key3和key4
	var val3 map[string]interface{}
	found, err = c.Get(ctx, "key3", &val3, fallback, nil)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "subvalue", val3["subkey"])

	// 刷新key3和key4
	refreshResults := make(map[string]interface{})
	assert.NoError(t, c.MRefresh(ctx, []string{"key3", "key4"}, &refreshResults, batchFallback, nil))
	assert.Equal(t, "value4", refreshResults["key4"])

	// 测试TTL
	opts := &cacher.CacheOptions{
		TTL: 100 * time.Millisecond,
	}

	// 设置带TTL的键
	var val4 string
	found, err = c.Get(ctx, "key4", &val4, fallback, opts)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "value4", val4)

	// 立即检查存在性
	var val4Again string
	found, err = c.Get(ctx, "key4", &val4Again, nil, nil)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "value4", val4Again)

	// 等待过期
	time.Sleep(150 * time.Millisecond)

	// 检查是否已过期
	var expiredVal string
	found, err = c.Get(ctx, "key4", &expiredVal, nil, nil)
	assert.NoError(t, err)
	assert.False(t, found)
}

// TestCacheImpl 测试CacheImpl实现
func TestCacheImpl(t *testing.T) {
	// 创建底层存储（使用gcache作为示例）
	cache := bluelegcache.New(100).LRU().Build()
	store := gcache.NewGCacheStore(cache)

	// 创建Cacher
	c := cacher.NewCacher(store)

	// 运行通用测试
	TestCacher(t, c)
}