package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/bluele/gcache"
	"github.com/dgraph-io/ristretto"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 通用的Store测试套件
func testStoreImplementation(t *testing.T, store Store) {
	ctx := context.Background()

	t.Run("Get_NotFound", func(t *testing.T) {
		var result string
		found, err := store.Get(ctx, "nonexistent", &result)
		assert.NoError(t, err)
		assert.False(t, found)
	})

	t.Run("Set_And_Get", func(t *testing.T) {
		// 设置数据
		items := map[string]interface{}{
			"string_key": "test_value",
			"int_key":    42,
		}
		err := store.MSet(ctx, items, 0)
		require.NoError(t, err)

		// 获取字符串
		var strResult string
		found, err := store.Get(ctx, "string_key", &strResult)
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "test_value", strResult)

		// 获取整数
		var intResult int
		found, err = store.Get(ctx, "int_key", &intResult)
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 42, intResult)
	})

	t.Run("MGet", func(t *testing.T) {
		// 设置测试数据
		items := map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		}
		err := store.MSet(ctx, items, 0)
		require.NoError(t, err)

		// 批量获取
		keys := []string{"key1", "key2", "key3"} // key3不存在
		result := make(map[string]string)
		err = store.MGet(ctx, keys, &result)
		assert.NoError(t, err)

		// 验证结果
		assert.Equal(t, "value1", result["key1"])
		assert.Equal(t, "value2", result["key2"])
		_, exists := result["key3"]
		assert.False(t, exists)
	})

	t.Run("Exists", func(t *testing.T) {
		// 设置测试数据
		items := map[string]interface{}{
			"existing_key": "value",
		}
		err := store.MSet(ctx, items, 0)
		require.NoError(t, err)

		// 检查存在性
		keys := []string{"existing_key", "nonexistent_key"}
		result, err := store.Exists(ctx, keys)
		assert.NoError(t, err)
		assert.True(t, result["existing_key"])
		assert.False(t, result["nonexistent_key"])
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
		deleted, err := store.Del(ctx, "del_key1", "del_key2", "nonexistent")
		assert.NoError(t, err)
		assert.Equal(t, int64(2), deleted)

		// 验证键已被删除
		var result string
		found, err := store.Get(ctx, "del_key1", &result)
		assert.NoError(t, err)
		assert.False(t, found)
	})

	t.Run("ComplexType", func(t *testing.T) {
		type User struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}

		user := User{ID: 123, Name: "John"}
		items := map[string]interface{}{
			"user_key": user,
		}
		err := store.MSet(ctx, items, 0)
		require.NoError(t, err)

		var result User
		found, err := store.Get(ctx, "user_key", &result)
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, user, result)
	})
}

func TestRedisStore(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	store := NewRedisStore(client)
	testStoreImplementation(t, store)
}

func TestRistrettoStore(t *testing.T) {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // 10M counters
		MaxCost:     1 << 30, // 1GB
		BufferItems: 64,      // 64 items buffer
	})
	require.NoError(t, err)

	store := NewRistrettoStore(cache)
	testStoreImplementation(t, store)
}

func TestGCacheStore(t *testing.T) {
	cache := gcache.New(1000).LRU().Build()
	store := NewGCacheStore(cache)
	testStoreImplementation(t, store)
}

// TTL测试（仅对支持TTL的存储进行测试）
func TestRedisStore_TTL(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	store := NewRedisStore(client)
	ctx := context.Background()

	// 设置带TTL的数据
	items := map[string]interface{}{
		"ttl_key": "ttl_value",
	}
	ttl := 2 * time.Second
	err = store.MSet(ctx, items, ttl)
	require.NoError(t, err)

	// 立即获取应该成功
	var result string
	found, err := store.Get(ctx, "ttl_key", &result)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "ttl_value", result)

	// 使用miniredis的FastForward来模拟时间流逝
	mr.FastForward(3 * time.Second)

	// 再次获取应该失败
	found, err = store.Get(ctx, "ttl_key", &result)
	assert.NoError(t, err)
	assert.False(t, found)
}