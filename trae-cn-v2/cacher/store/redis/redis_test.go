package redis

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"go-cache/cacher/store"
)

// TestRedisStore 测试RedisStore实现
func TestRedisStore(t *testing.T) {
	// 启动miniredis服务器
	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("启动miniredis失败: %v", err)
	}
	defer server.Close()

	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr: server.Addr(),
	})

	// 测试Store接口
	store.StoreTestHelper(t, func() (store.Store, func(), error) {
		// 创建RedisStore实例
		redisStore := NewRedisStoreWithClient(client)

		// 返回Store实例和清理函数
		return redisStore, func() {
			// 清理函数
			redisStore.Close()
		}, nil
	})
}