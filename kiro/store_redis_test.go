package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRedisStore(t *testing.T) (Store, *miniredis.Miniredis) {
	// 启动miniredis
	mr, err := miniredis.Run()
	require.NoError(t, err)

	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	store := NewRedisStore(client)
	return store, mr
}

func TestRedisStore_Get_Success(t *testing.T) {
	store, mr := setupRedisStore(t)
	defer mr.Close()
	ctx := context.Background()

	// 预设数据
	mr.Set("test_key", `"test_value"`)

	var result string
	found, err := store.Get(ctx, "test_key", &result)

	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "test_value", result)
}

func TestRedisStore_Get_NotFound(t *testing.T) {
	store, mr := setupRedisStore(t)
	defer mr.Close()
	ctx := context.Background()

	var result string
	found, err := store.Get(ctx, "nonexistent_key", &result)

	assert.NoError(t, err)
	assert.False(t, found)
}

func TestRedisStore_Get_ComplexType(t *testing.T) {
	store, mr := setupRedisStore(t)
	defer mr.Close()
	ctx := context.Background()

	type User struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	// 预设复杂类型数据
	mr.Set("user_key", `{"id":123,"name":"John"}`)

	var result User
	found, err := store.Get(ctx, "user_key", &result)

	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, User{ID: 123, Name: "John"}, result)
}

func TestRedisStore_MGet_Success(t *testing.T) {
	store, mr := setupRedisStore(t)
	defer mr.Close()
	ctx := context.Background()

	// 预设数据
	mr.Set("key1", `"value1"`)
	mr.Set("key2", `"value2"`)

	keys := []string{"key1", "key2", "key3"} // key3不存在
	result := make(map[string]string)

	err := store.MGet(ctx, keys, &result)

	assert.NoError(t, err)
	assert.Equal(t, map[string]string{
		"key1": "value1",
		"key2": "value2",
	}, result)
}

func TestRedisStore_MGet_EmptyKeys(t *testing.T) {
	store, mr := setupRedisStore(t)
	defer mr.Close()
	ctx := context.Background()

	result := make(map[string]string)
	err := store.MGet(ctx, []string{}, &result)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestRedisStore_Exists_Success(t *testing.T) {
	store, mr := setupRedisStore(t)
	defer mr.Close()
	ctx := context.Background()

	// 预设数据
	mr.Set("existing_key", "value")

	keys := []string{"existing_key", "nonexistent_key"}
	result, err := store.Exists(ctx, keys)

	assert.NoError(t, err)
	assert.Equal(t, map[string]bool{
		"existing_key":    true,
		"nonexistent_key": false,
	}, result)
}

func TestRedisStore_MSet_Success(t *testing.T) {
	store, mr := setupRedisStore(t)
	defer mr.Close()
	ctx := context.Background()

	items := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
		"key3": map[string]string{"nested": "value"},
	}

	err := store.MSet(ctx, items, 0)
	assert.NoError(t, err)

	// 验证数据是否正确设置
	val1, _ := mr.Get("key1")
	assert.Equal(t, `"value1"`, val1)
	val2, _ := mr.Get("key2")
	assert.Equal(t, "123", val2)
	val3, _ := mr.Get("key3")
	assert.Equal(t, `{"nested":"value"}`, val3)
}

func TestRedisStore_MSet_WithTTL(t *testing.T) {
	store, mr := setupRedisStore(t)
	defer mr.Close()
	ctx := context.Background()

	items := map[string]interface{}{
		"key1": "value1",
	}

	ttl := 10 * time.Second
	err := store.MSet(ctx, items, ttl)
	assert.NoError(t, err)

	// 验证TTL是否设置
	assert.True(t, mr.TTL("key1") > 0)
	assert.True(t, mr.TTL("key1") <= ttl)
}

func TestRedisStore_Del_Success(t *testing.T) {
	store, mr := setupRedisStore(t)
	defer mr.Close()
	ctx := context.Background()

	// 预设数据
	mr.Set("key1", "value1")
	mr.Set("key2", "value2")

	deleted, err := store.Del(ctx, "key1", "key2", "key3") // key3不存在

	assert.NoError(t, err)
	assert.Equal(t, int64(2), deleted)
	
	// 验证键是否被删除
	assert.False(t, mr.Exists("key1"))
	assert.False(t, mr.Exists("key2"))
}

func TestRedisStore_Del_EmptyKeys(t *testing.T) {
	store, mr := setupRedisStore(t)
	defer mr.Close()
	ctx := context.Background()

	deleted, err := store.Del(ctx)

	assert.NoError(t, err)
	assert.Equal(t, int64(0), deleted)
}