package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/bluele/gcache"
	"github.com/dgraph-io/ristretto"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type storeTestCase struct {
	name  string
	store Store
}

func TestStore(t *testing.T) {
	// 测试Redis Store
	miniredis := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{
		Addr: miniredis.Addr(),
	})
	defer rdb.Close()

	// 测试Ristretto Store
	rcache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // number of keys to track frequency of (10M).
		MaxCost:     1 << 30, // maximum cost of cache (1GB).
		BufferItems: 64,      // number of keys per Get buffer.
	})
	require.NoError(t, err)

	// 测试GCache Store
	gcache := gcache.New(1000).LRU().Build()

	testCases := []storeTestCase{
		{"Redis", NewRedisStore(rdb)},
		{"Ristretto", NewRistrettoStore(rcache)},
		{"GCache", NewGCacheStore(gcache)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testStoreBasic(t, tc.store)
			testStoreMGet(t, tc.store)
			testStoreExists(t, tc.store)
			testStoreMSet(t, tc.store)
			testStoreDel(t, tc.store)
		})
	}
}

func testStoreBasic(t *testing.T, store Store) {
	ctx := context.Background()

	// 测试基本Set/Get
	var value string
	found, err := store.Get(ctx, "key1", &value)
	assert.NoError(t, err)
	assert.False(t, found)

	// 设置值
	items := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}
	err = store.MSet(ctx, items, 0)
	assert.NoError(t, err)

	// 验证获取
	found, err = store.Get(ctx, "key1", &value)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "value1", value)

	var intValue int
	found, err = store.Get(ctx, "key2", &intValue)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, 123, intValue)

	var boolValue bool
	found, err = store.Get(ctx, "key3", &boolValue)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.True(t, boolValue)
}

func testStoreMGet(t *testing.T, store Store) {
	ctx := context.Background()

	// 设置测试数据
	items := map[string]interface{}{
		"mget1": "value1",
		"mget2": "value2",
		"mget3": 42,
	}
	err := store.MSet(ctx, items, 0)
	assert.NoError(t, err)

	// 测试MGet
	result := make(map[string]string)
	err = store.MGet(ctx, []string{"mget1", "mget2", "mget3", "nonexistent"}, &result)
	assert.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, "value1", result["mget1"])
	assert.Equal(t, "value2", result["mget2"])
	assert.Equal(t, "42", result["mget3"])
}

func testStoreExists(t *testing.T, store Store) {
	ctx := context.Background()

	// 设置测试数据
	items := map[string]interface{}{
		"exists1": "value1",
		"exists2": "value2",
	}
	err := store.MSet(ctx, items, 0)
	assert.NoError(t, err)

	// 测试Exists
	exists, err := store.Exists(ctx, []string{"exists1", "exists2", "nonexistent"})
	assert.NoError(t, err)
	assert.Len(t, exists, 3)
	assert.True(t, exists["exists1"])
	assert.True(t, exists["exists2"])
	assert.False(t, exists["nonexistent"])
}

func testStoreMSet(t *testing.T, store Store) {
	ctx := context.Background()

	// 测试批量设置
	items := map[string]interface{}{
		"mset1": "string value",
		"mset2": 12345,
		"mset3": []int{1, 2, 3, 4, 5},
		"mset4": map[string]interface{}{
			"nested": "value",
		},
	}
	err := store.MSet(ctx, items, 0)
	assert.NoError(t, err)

	// 验证所有值
	var strValue string
	found, err := store.Get(ctx, "mset1", &strValue)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "string value", strValue)

	var intValue int
	found, err = store.Get(ctx, "mset2", &intValue)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, 12345, intValue)

	var sliceValue []int
	found, err = store.Get(ctx, "mset3", &sliceValue)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, []int{1, 2, 3, 4, 5}, sliceValue)

	var mapValue map[string]interface{}
	found, err = store.Get(ctx, "mset4", &mapValue)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "value", mapValue["nested"])
}

func testStoreDel(t *testing.T, store Store) {
	ctx := context.Background()

	// 设置测试数据
	items := map[string]interface{}{
		"del1": "value1",
		"del2": "value2",
		"del3": "value3",
	}
	err := store.MSet(ctx, items, 0)
	assert.NoError(t, err)

	// 验证存在
	exists, err := store.Exists(ctx, []string{"del1", "del2", "del3"})
	assert.NoError(t, err)
	assert.True(t, exists["del1"])
	assert.True(t, exists["del2"])
	assert.True(t, exists["del3"])

	// 删除部分键
	deleted, err := store.Del(ctx, "del1", "del2")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), deleted)

	// 验证删除结果
	exists, err = store.Exists(ctx, []string{"del1", "del2", "del3"})
	assert.NoError(t, err)
	assert.False(t, exists["del1"])
	assert.False(t, exists["del2"])
	assert.True(t, exists["del3"])
}

func TestRedisStoreWithTTL(t *testing.T) {
	miniredis := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{
		Addr: miniredis.Addr(),
	})
	defer rdb.Close()

	store := NewRedisStore(rdb)
	ctx := context.Background()

	// 测试TTL
	items := map[string]interface{}{
		"ttl1": "value1",
		"ttl2": "value2",
	}
	err := store.MSet(ctx, items, 100*time.Millisecond)
	assert.NoError(t, err)

	// 立即验证存在
	exists, err := store.Exists(ctx, []string{"ttl1", "ttl2"})
	assert.NoError(t, err)
	assert.True(t, exists["ttl1"])
	assert.True(t, exists["ttl2"])

	// 等待过期（miniredis支持时间快进）
	miniredis.FastForward(200 * time.Millisecond)

	// 验证已过期
	exists, err = store.Exists(ctx, []string{"ttl1", "ttl2"})
	assert.NoError(t, err)
	assert.False(t, exists["ttl1"])
	assert.False(t, exists["ttl2"])
}