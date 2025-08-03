package cacher

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-cache/cacher/store/redis"
	"go-cache/cacher/store/ristretto"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

// 测试数据结构
type TestUser struct {
	ID   int
	Name string
	Age  int
}

// 创建Redis存储实例用于测试
func createRedisStore(t *testing.T) (*redis.Store, func()) {
	// 创建一个miniredis实例用于测试
	s, err := miniredis.Run()
	require.NoError(t, err)

	// 创建Redis客户端连接到miniredis
	client := goredis.NewClient(&goredis.Options{
		Addr: s.Addr(),
	})

	// 创建Redis存储实例
	redisStore, err := redis.New(redis.Options{
		Client: client,
	})
	require.NoError(t, err)

	// 返回清理函数
	cleanup := func() {
		s.Close()
	}

	return redisStore, cleanup
}

// 创建Ristretto存储实例用于测试
func createRistrettoStore() (*ristretto.Store, error) {
	// 创建Ristretto存储实例
	return ristretto.New(ristretto.Options{
		NumCounters: 1e4,
		MaxCost:     1e6,
		BufferItems: 64,
	})
}

// TestDefaultCacherWithRedis 测试使用Redis作为存储后端的DefaultCacher
func TestDefaultCacherWithRedis(t *testing.T) {
	redisStore, cleanup := createRedisStore(t)
	defer cleanup()

	// 创建DefaultCacher实例
	cacher := NewDefaultCacher[TestUser](DefaultCacherOptions{
		Store:      redisStore,
		Prefix:     "test",
		DefaultTTL: time.Minute,
	})

	// 运行通用测试
	testDefaultCacher(t, cacher)
}

// TestDefaultCacherWithRistretto 测试使用Ristretto作为存储后端的DefaultCacher
func TestDefaultCacherWithRistretto(t *testing.T) {
	ristrettoStore, err := createRistrettoStore()
	require.NoError(t, err)
	defer ristrettoStore.Close()

	// 创建DefaultCacher实例
	cacher := NewDefaultCacher[TestUser](DefaultCacherOptions{
		Store:      ristrettoStore,
		Prefix:     "test",
		DefaultTTL: time.Minute,
	})

	// 运行通用测试
	testDefaultCacher(t, cacher)
}

// TestNewDefaultCacher 测试创建DefaultCacher时的各种情况
func TestNewDefaultCacherError(t *testing.T) {
	// 测试没有提供Store的情况
	assert.Panics(t, func() {
		NewDefaultCacher[TestUser](DefaultCacherOptions{
			Prefix:     "test",
			DefaultTTL: time.Minute,
		})
	})

	// 测试使用默认TTL的情况
	redisStore, cleanup := createRedisStore(t)
	defer cleanup()

	defaultCacher := NewDefaultCacher[TestUser](DefaultCacherOptions{
		Store:  redisStore,
		Prefix: "test",
		// 不设置DefaultTTL，应该使用默认值
	})
	assert.NotNil(t, defaultCacher)
	assert.Equal(t, time.Hour, defaultCacher.defaultTTL)
	
	// 测试使用Redis存储
	redisCacher := NewDefaultCacher[TestUser](DefaultCacherOptions{
		Store:      redisStore,
		Prefix:     "test",
		DefaultTTL: time.Minute,
	})
	assert.NotNil(t, redisCacher)
	
	// 测试使用Ristretto存储
	ristrettoStore, err := createRistrettoStore()
	require.NoError(t, err)
	defer ristrettoStore.Close()
	
	ristrettoCacher := NewDefaultCacher[TestUser](DefaultCacherOptions{
		Store:      ristrettoStore,
		Prefix:     "test",
		DefaultTTL: time.Minute,
	})
	assert.NotNil(t, ristrettoCacher)
}

// testDefaultCacher 通用的DefaultCacher测试函数
func testDefaultCacher(t *testing.T, cacher *DefaultCacher[TestUser]) {
	ctx := context.Background()

	// 测试Get方法
	t.Run("Get", func(t *testing.T) {
		// 测试缓存未命中且没有fallback的情况
		var unknownUser TestUser
		found, err := cacher.Get(ctx, "unknown", &unknownUser, nil, nil)
		require.Error(t, err)
		assert.False(t, found)

		// 测试缓存未命中但有fallback的情况
		user := TestUser{ID: 1, Name: "Alice", Age: 30}
		fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
			return user, true, nil
		}

		var resultUser TestUser
		found, err = cacher.Get(ctx, "user:1", &resultUser, fallback, nil)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, user, resultUser)

		// 测试缓存命中的情况
		var cachedUser TestUser
		found, err = cacher.Get(ctx, "user:1", &cachedUser, nil, nil)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, user, cachedUser)

		// 测试fallback返回错误的情况
		fallbackErr := func(ctx context.Context, key string) (interface{}, bool, error) {
			return nil, false, errors.New("fallback error")
		}

		var errorUser TestUser
		found, err = cacher.Get(ctx, "error", &errorUser, fallbackErr, nil)
		require.Error(t, err)
		assert.False(t, found)
		assert.Contains(t, err.Error(), "fallback error")
	})

	// 测试MGet方法
	t.Run("MGet", func(t *testing.T) {
		// 准备测试数据
		users := map[string]TestUser{
			"user:2": {ID: 2, Name: "Bob", Age: 25},
			"user:3": {ID: 3, Name: "Charlie", Age: 35},
		}

		// 先将数据存入缓存
		for key, user := range users {
			var resultUser TestUser
			found, err := cacher.Get(ctx, key, &resultUser, func(ctx context.Context, key string) (interface{}, bool, error) {
				return user, true, nil
			}, nil)
			require.NoError(t, err)
			assert.True(t, found)
		}

		// 测试批量获取
		resultMap := make(map[string]*TestUser)
		resultMap["user:2"] = &TestUser{}
		resultMap["user:3"] = &TestUser{}
		
		foundKeys, err := cacher.MGet(ctx, []string{"user:2", "user:3"}, &resultMap, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, 2, len(foundKeys))
		assert.Contains(t, foundKeys, "user:2")
		assert.Contains(t, foundKeys, "user:3")
		assert.Equal(t, users["user:2"], *resultMap["user:2"])
		assert.Equal(t, users["user:3"], *resultMap["user:3"])

		// 测试部分缓存未命中且有batchFallback的情况
		batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, []string, error) {
			result := make(map[string]interface{})
			foundKeys := []string{}
			for _, key := range keys {
				if key == "user:4" {
					result[key] = TestUser{ID: 4, Name: "Dave", Age: 40}
					foundKeys = append(foundKeys, key)
				}
			}
			return result, foundKeys, nil
		}

		resultMap = make(map[string]*TestUser)
		resultMap["user:2"] = &TestUser{}
		resultMap["user:4"] = &TestUser{}
		
		foundKeys, err = cacher.MGet(ctx, []string{"user:2", "user:4"}, &resultMap, batchFallback, nil)
		require.NoError(t, err)
		assert.Equal(t, 2, len(foundKeys))
		assert.Contains(t, foundKeys, "user:2")
		assert.Contains(t, foundKeys, "user:4")
		assert.Equal(t, users["user:2"], *resultMap["user:2"])
		assert.Equal(t, TestUser{ID: 4, Name: "Dave", Age: 40}, *resultMap["user:4"])

		// 测试缓存未命中且没有batchFallback的情况
		resultMap = make(map[string]*TestUser)
		resultMap["user:5"] = &TestUser{}
		foundKeys, err = cacher.MGet(ctx, []string{"user:5"}, &resultMap, nil, nil)
		require.Error(t, err)
		assert.Empty(t, foundKeys)

		// 测试batchFallback返回错误的情况
		batchFallbackErr := func(ctx context.Context, keys []string) (map[string]interface{}, []string, error) {
			return nil, nil, errors.New("batch fallback error")
		}

		resultMap = make(map[string]*TestUser)
		resultMap["user:5"] = &TestUser{}
		foundKeys, err = cacher.MGet(ctx, []string{"user:5"}, resultMap, batchFallbackErr, nil)
		require.Error(t, err)
		assert.Empty(t, foundKeys)
		assert.Contains(t, err.Error(), "batch fallback error")

		// 测试空键列表
		resultMap = make(map[string]*TestUser)
		foundKeys, err = cacher.MGet(ctx, []string{}, resultMap, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, 0, len(foundKeys))
	})

	// 测试MDelete方法
	t.Run("MDelete", func(t *testing.T) {
		// 准备测试数据
		user := TestUser{ID: 5, Name: "Eve", Age: 28}
		var resultUser TestUser
		found, err := cacher.Get(ctx, "user:5", &resultUser, func(ctx context.Context, key string) (interface{}, bool, error) {
			return user, true, nil
		}, nil)
		require.NoError(t, err)
		assert.True(t, found)

		// 验证数据已存入缓存
		var cachedUser TestUser
		found, err = cacher.Get(ctx, "user:5", &cachedUser, nil, nil)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, user, cachedUser)

		// 删除缓存
		deleted, err := cacher.MDelete(ctx, []string{"user:5"})
		require.NoError(t, err)
		assert.Equal(t, int64(1), deleted)

		// 验证缓存已删除
		var deletedUser TestUser
		found, err = cacher.Get(ctx, "user:5", &deletedUser, nil, nil)
		require.Error(t, err)
		assert.False(t, found)

		// 测试空键列表
		deleted, err = cacher.MDelete(ctx, []string{})
		require.NoError(t, err)
		assert.Equal(t, int64(0), deleted)
	})

	// 测试MRefresh方法
	t.Run("MRefresh", func(t *testing.T) {
		// 准备测试数据
		oldUser := TestUser{ID: 6, Name: "Frank", Age: 45}
		var resultUser TestUser
		found, err := cacher.Get(ctx, "user:6", &resultUser, func(ctx context.Context, key string) (interface{}, bool, error) {
			return oldUser, true, nil
		}, nil)
		require.NoError(t, err)
		assert.True(t, found)

		// 刷新缓存
		newUser := TestUser{ID: 6, Name: "Frank", Age: 46} // 年龄增加
		batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, []string, error) {
			result := make(map[string]interface{})
			foundKeys := []string{}
			for _, key := range keys {
				if key == "user:6" {
					result[key] = newUser
					foundKeys = append(foundKeys, key)
				}
			}
			return result, foundKeys, nil
		}

		refreshMap := make(map[string]*TestUser)
		refreshMap["user:6"] = &TestUser{}
		err = cacher.MRefresh(ctx, []string{"user:6"}, &refreshMap, batchFallback, nil)
		require.NoError(t, err)

		// 验证缓存已刷新
		var cachedUser TestUser
		found, err = cacher.Get(ctx, "user:6", &cachedUser, nil, nil)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, newUser, cachedUser)

		// 测试没有batchFallback的情况
		refreshMap = make(map[string]*TestUser)
		refreshMap["user:6"] = &TestUser{}
		err = cacher.MRefresh(ctx, []string{"user:6"}, &refreshMap, nil, nil)
		require.Error(t, err)

		// 测试batchFallback返回错误的情况
		batchFallbackErr := func(ctx context.Context, keys []string) (map[string]interface{}, []string, error) {
			return nil, nil, errors.New("refresh error")
		}

		refreshMap = make(map[string]*TestUser)
		refreshMap["user:6"] = &TestUser{}
		err = cacher.MRefresh(ctx, []string{"user:6"}, &refreshMap, batchFallbackErr, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "refresh error")

		// 测试空键列表
		refreshMap = make(map[string]*TestUser)
		err = cacher.MRefresh(ctx, []string{}, &refreshMap, batchFallback, nil)
		require.NoError(t, err)
	})

	// 测试自定义TTL
	t.Run("CustomTTL", func(t *testing.T) {
		// 使用自定义TTL
		user := TestUser{ID: 7, Name: "Grace", Age: 50}
		var resultUser TestUser
		opts := &CacheOptions{TTL: 2 * time.Second}
		found, err := cacher.Get(ctx, "user:7", &resultUser, func(ctx context.Context, key string) (interface{}, bool, error) {
			return user, true, nil
		}, opts)
		require.NoError(t, err)
		assert.True(t, found)

		// 立即获取应该存在
		var cachedUser TestUser
		found, err = cacher.Get(ctx, "user:7", &cachedUser, nil, nil)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, user, cachedUser)

		// 等待TTL过期（这里假设存储后端正确实现了TTL）
		// 注意：这个测试可能不适用于所有存储后端，特别是在单元测试中
		// 对于Redis，我们可以使用miniredis的FastForward方法模拟时间流逝
		if _, ok := cacher.store.(*redis.Store); ok {
			// 尝试获取底层的miniredis实例并快进时间
			// 这里只是一个示例，实际上可能需要更复杂的处理
			// 在实际测试中，我们可能需要为每个存储后端提供特定的测试方法
		}
	})
}