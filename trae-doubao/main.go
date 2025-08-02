package main

import (
	"context"
	"fmt"
	"time"

	"github.com/xiumu/git/me/go-cache/trae/pkg/cacher"
	"github.com/xiumu/git/me/go-cache/trae/pkg/store/redis"
	"github.com/xiumu/git/me/go-cache/trae/pkg/store/ristretto"
	"github.com/xiumu/git/me/go-cache/trae/pkg/store/gcache"
	goredis "github.com/go-redis/redis/v8"
	dgraphristretto "github.com/dgraph-io/ristretto"
	bluelegcache "github.com/bluele/gcache"
	"github.com/alicebob/miniredis/v2"
)

func main() {
	ctx := context.Background()

	// 示例1：使用Redis作为存储后端
	fmt.Println("=== 使用Redis作为存储后端 ===")
	// 启动mini redis服务器
	mr, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer mr.Close()

	// 创建Redis客户端
	redisClient := goredis.NewClient(&goredis.Options{
		Addr: mr.Addr(),
	})

	// 创建RedisStore
	redisStore := redis.NewRedisStore(redisClient)

	// 创建Cacher
	redisCacher := cacher.NewCacher(redisStore)

	// 使用Cacher
	var val string
	found, err := redisCacher.Get(ctx, "redis_key", &val, func(ctx context.Context, key string) (interface{}, bool, error) {
		return "redis_value", true, nil
	}, nil)
	fmt.Printf("Get result: found=%v, val=%v, err=%v\n", found, val, err)

	// 示例2：使用Ristretto作为存储后端
	fmt.Println("\n=== 使用Ristretto作为存储后端 ===")
	// 创建Ristretto缓存
	ristrettoCache, err := dgraphristretto.NewCache(&dgraphristretto.Config{
		NumCounters: 1000,
		MaxCost:     100000,
		BufferItems: 64,
	})
	if err != nil {
		panic(err)
	}

	// 创建RistrettoStore
	ristrettoStore := ristretto.NewRistrettoStore(ristrettoCache)

	// 创建Cacher
	ristrettoCacher := cacher.NewCacher(ristrettoStore)

	// 使用Cacher
	var val2 int
	found, err = ristrettoCacher.Get(ctx, "ristretto_key", &val2, func(ctx context.Context, key string) (interface{}, bool, error) {
		return 42, true, nil
	}, nil)
	fmt.Printf("Get result: found=%v, val=%v, err=%v\n", found, val2, err)

	// 示例3：使用GCache作为存储后端
	fmt.Println("\n=== 使用GCache作为存储后端 ===")
	// 创建GCache缓存
	gcacheCache := bluelegcache.New(100).LRU().Build()

	// 创建GCacheStore
	gcacheStore := gcache.NewGCacheStore(gcacheCache)

	// 创建Cacher
	gcacheCacher := cacher.NewCacher(gcacheStore)

	// 使用Cacher
	var val3 map[string]interface{}
	found, err = gcacheCacher.Get(ctx, "gcache_key", &val3, func(ctx context.Context, key string) (interface{}, bool, error) {
		return map[string]interface{}{"subkey": "subvalue"}, true, nil
	}, nil)
	fmt.Printf("Get result: found=%v, val=%v, err=%v\n", found, val3, err)

	// 示例4：测试批量操作
	fmt.Println("\n=== 测试批量操作 ===")
	// 批量获取
	results := make(map[string]interface{})
	keys := []string{"key1", "key2", "key3"}
	err = gcacheCacher.MGet(ctx, keys, &results, func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		result := make(map[string]interface{})
		for _, key := range keys {
			if key == "key1" {
				result[key] = "value1"
			}
			if key == "key2" {
				result[key] = 100
			}
			if key == "key3" {
				result[key] = true
			}
		}
		return result, nil
	}, nil)
	fmt.Printf("MGet result: %v, err=%v\n", results, err)

	// 批量删除
	count, err := gcacheCacher.MDelete(ctx, []string{"key1", "key2"})
	fmt.Printf("MDelete result: count=%v, err=%v\n", count, err)

	// 批量刷新
	refreshResults := make(map[string]interface{})
	err = gcacheCacher.MRefresh(ctx, []string{"key3"}, &refreshResults, func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		return map[string]interface{}{"key3": false}, nil
	}, nil)
	fmt.Printf("MRefresh result: %v, err=%v\n", refreshResults, err)

	// 示例5：测试TTL
	fmt.Println("\n=== 测试TTL ===")
	opts := &cacher.CacheOptions{
		TTL: 1 * time.Second,
	}

	var ttlVal string
	found, err = gcacheCacher.Get(ctx, "ttl_key", &ttlVal, func(ctx context.Context, key string) (interface{}, bool, error) {
		return "ttl_value", true, nil
	}, opts)
	fmt.Printf("Get TTL key result: found=%v, val=%v, err=%v\n", found, ttlVal, err)

	// 立即检查
	var ttlValAgain string
	found, err = gcacheCacher.Get(ctx, "ttl_key", &ttlValAgain, nil, nil)
	fmt.Printf("Get TTL key immediately: found=%v, val=%v, err=%v\n", found, ttlValAgain, err)

	// 等待过期
	time.Sleep(1500 * time.Millisecond)

	// 检查是否过期
	var expiredVal string
	found, err = gcacheCacher.Get(ctx, "ttl_key", &expiredVal, nil, nil)
	fmt.Printf("Get TTL key after expiration: found=%v, val=%v, err=%v\n", found, expiredVal, err)
}