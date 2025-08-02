package redis

import (
	"context"
	"testing"
	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
)

func TestRedisStore(t *testing.T) {
	// 启动miniredis服务器
	server, err := miniredis.Run()
	assert.NoError(t, err)
	defer server.Close()

	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr: server.Addr(),
	})
	defer client.Close()

	store := NewRedisStore(client)
	ctx := context.Background()

	// 测试Exists方法
	result, err := store.Exists(ctx, []string{"key1", "key2"})
	assert.NoError(t, err)
	assert.False(t, result["key1"])
	assert.False(t, result["key2"])

	// 设置一些键值对
	server.Set("key1", "value1")
	server.Set("key2", "value2")

	// 再次测试Exists方法
	result, err = store.Exists(ctx, []string{"key1", "key2"})
	assert.NoError(t, err)
	assert.True(t, result["key1"])
	assert.True(t, result["key2"])

	// 测试Del方法
	deleted, err := store.Del(ctx, "key1", "key2", "key3")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), deleted)
}