package test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/xiumu/git/me/go-cache/trae/pkg/store"
	"github.com/xiumu/git/me/go-cache/trae/pkg/store/redis"
	"github.com/xiumu/git/me/go-cache/trae/pkg/store/ristretto"
	"github.com/xiumu/git/me/go-cache/trae/pkg/store/gcache"
	goredis "github.com/go-redis/redis/v8"
	"github.com/alicebob/miniredis/v2"
	dgraphristretto "github.com/dgraph-io/ristretto"
	bluelegcache "github.com/bluele/gcache"
)

// TestStore 测试Store接口的通用测试函数
func TestStore(t *testing.T, s store.Store) {
	ctx := context.Background()

	// 测试Set和Get
	items := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": map[string]interface{}{
			"subkey": "subvalue",
		},
	}

	// 设置值
	assert.NoError(t, s.MSet(ctx, items, 0))

	// 检查存在性
	exists, err := s.Exists(ctx, []string{"key1", "key2", "key3", "key4"})
	assert.NoError(t, err)
	assert.True(t, exists["key1"])
	assert.True(t, exists["key2"])
	assert.True(t, exists["key3"])
	assert.False(t, exists["key4"])

	// 获取单个值
	var val1 string
	found, err := s.Get(ctx, "key1", &val1)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "value1", val1)

	var val2 int
	found, err = s.Get(ctx, "key2", &val2)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, 42, val2)

	var val3 map[string]interface{}
	found, err = s.Get(ctx, "key3", &val3)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "subvalue", val3["subkey"])

	// 批量获取值
	results := make(map[string]interface{})
	assert.NoError(t, s.MGet(ctx, []string{"key1", "key2", "key3", "key4"}, &results))
	assert.Equal(t, "value1", results["key1"])
	assert.Equal(t, 42, results["key2"])
	assert.Equal(t, map[string]interface{}{"subkey": "subvalue"}, results["key3"])
	assert.Nil(t, results["key4"])

	// 测试删除
	count, err := s.Del(ctx, []string{"key1", "key2"})
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// 检查是否已删除
	exists, err = s.Exists(ctx, []string{"key1", "key2", "key3"})
	assert.NoError(t, err)
	assert.False(t, exists["key1"])
	assert.False(t, exists["key2"])
	assert.True(t, exists["key3"])

	// 测试TTL
	assert.NoError(t, s.MSet(ctx, map[string]interface{}{"key4": "value4"}, 100*time.Millisecond))

	// 立即检查存在性
	exists, err = s.Exists(ctx, []string{"key4"})
	assert.NoError(t, err)
	assert.True(t, exists["key4"])

	// 等待过期
	time.Sleep(150 * time.Millisecond)

	// 检查是否已过期
	exists, err = s.Exists(ctx, []string{"key4"})
	assert.NoError(t, err)
	assert.False(t, exists["key4"])
}

// TestRedisStore 测试RedisStore实现
func TestRedisStore(t *testing.T) {
	// 启动mini redis服务器
	mr, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer mr.Close()

	// 创建Redis客户端
	client := goredis.NewClient(&goredis.Options{
		Addr: mr.Addr(),
	})

	// 创建RedisStore
	s := redis.NewRedisStore(client)

	// 运行通用测试
	TestStore(t, s)
}

// TestRistrettoStore 测试RistrettoStore实现
func TestRistrettoStore(t *testing.T) {
	// 创建Ristretto缓存
	cache, err := dgraphristretto.NewCache(&dgraphristretto.Config{
		NumCounters: 1000,
		MaxCost:     100000,
		BufferItems: 64,
	})
	if err != nil {
		panic(err)
	}

	// 创建RistrettoStore
	s := ristretto.NewRistrettoStore(cache)

	// 运行通用测试
	TestStore(t, s)
}

// TestGCacheStore 测试GCacheStore实现
func TestGCacheStore(t *testing.T) {
	// 创建GCache缓存
	cache := bluelegcache.New(100).LRU().Build()

	// 创建GCacheStore
	s := gcache.NewGCacheStore(cache)

	// 运行通用测试
	TestStore(t, s)
}