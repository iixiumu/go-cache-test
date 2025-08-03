package redis

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"go-cache/cacher/store"
)

func TestRedisStore(t *testing.T) {
	// 创建一个miniredis实例用于测试
	s, err := miniredis.Run()
	require.NoError(t, err)
	defer s.Close()

	// 创建Redis客户端连接到miniredis
	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	// 创建Redis存储实例
	redisStore, err := New(Options{
		Client: client,
	})
	require.NoError(t, err)

	// 创建测试套件
	testSuite := &store.StoreTestSuite{
		NewStore: func() store.Store {
			return redisStore
		},
		Cleanup: func() {
			// 清空Redis数据库
			s.FlushAll()
		},
	}

	// 运行所有测试
	testSuite.RunTestSuite(t)
}

func TestRedisStoreWithOptions(t *testing.T) {
	// 创建一个miniredis实例用于测试
	s, err := miniredis.Run()
	require.NoError(t, err)
	defer s.Close()

	// 使用RedisOptions创建Redis存储实例
	redisStore, err := New(Options{
		RedisOptions: &redis.UniversalOptions{
			Addrs: []string{s.Addr()},
		},
	})
	require.NoError(t, err)

	// 创建测试套件
	testSuite := &store.StoreTestSuite{
		NewStore: func() store.Store {
			return redisStore
		},
		Cleanup: func() {
			// 清空Redis数据库
			s.FlushAll()
		},
	}

	// 运行所有测试
	testSuite.RunTestSuite(t)
}

func TestNewRedisStoreError(t *testing.T) {
	// 测试没有提供选项的情况
	_, err := New(Options{})
	require.Error(t, err)

	// 测试连接错误的情况
	_, err = New(Options{
		RedisOptions: &redis.UniversalOptions{
			Addrs: []string{"invalid:6379"},
		},
	})
	require.Error(t, err)
}