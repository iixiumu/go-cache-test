package redis

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"go-cache/cacher/store"
)

func TestRedisStore(t *testing.T) {
	// 启动miniredis
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	// 创建Redis Store
	redisStore := NewStore(client)

	// 运行通用测试套件
	testHelper := store.NewTestHelper(t, redisStore)
	testHelper.RunAllTests()
}
