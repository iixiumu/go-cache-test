package main

import (
	"context"
	"fmt"
	"time"

	cacher "go-cache"
	"go-cache/store/gcache"
	"go-cache/store/redis"
	"go-cache/store/ristretto"

	redisclient "github.com/redis/go-redis/v9"
)

func main() {
	// 创建不同的存储后端
	// 1. Redis 存储
	redisClient := redisclient.NewClient(&redisclient.Options{
		Addr: "localhost:6379",
	})
	redisStore := redis.NewRedisStore(redisClient)

	// 2. Ristretto 存储
	ristrettoStore, err := ristretto.NewRistrettoStore(&ristretto.Config{
		NumCounters: 1000,
		MaxCost:     1000,
		BufferItems: 64,
	})
	if err != nil {
		panic(err)
	}
	defer ristrettoStore.Close()

	// 3. GCache 存储
	gcacheStore, err := gcache.NewGCacheStore(&gcache.Config{
		Size: 1000,
	})
	if err != nil {
		panic(err)
	}
	defer gcacheStore.Close()

	// 创建不同的缓存实例
	redisCacher := cacher.NewCacher(redisStore)
	ristrettoCacher := cacher.NewCacher(ristrettoStore)
	gcacheCacher := cacher.NewCacher(gcacheStore)

	// 定义回退函数
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		// 模拟从数据库或其他数据源获取数据
		fmt.Printf("从数据源获取数据: %s\n", key)
		return fmt.Sprintf("value_for_%s", key), true, nil
	}

	// 定义批量回退函数
	batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		// 模拟从数据库或其他数据源批量获取数据
		fmt.Printf("从数据源批量获取数据: %v\n", keys)
		result := make(map[string]interface{})
		for _, key := range keys {
			result[key] = fmt.Sprintf("value_for_%s", key)
		}
		return result, nil
	}

	// 使用 Redis 缓存
	fmt.Println("=== 使用 Redis 缓存 ===")
	ctx := context.Background()
	var value string
	found, err := redisCacher.Get(ctx, "key1", &value, fallback, &cacher.CacheOptions{TTL: time.Minute})
	if err != nil {
		panic(err)
	}
	if found {
		fmt.Printf("获取到值: %s\n", value)
	}

	// 使用 Ristretto 缓存
	fmt.Println("\n=== 使用 Ristretto 缓存 ===")
	var value2 string
	found, err = ristrettoCacher.Get(ctx, "key2", &value2, fallback, &cacher.CacheOptions{TTL: time.Minute})
	if err != nil {
		panic(err)
	}
	if found {
		fmt.Printf("获取到值: %s\n", value2)
	}

	// 使用 GCache 缓存
	fmt.Println("\n=== 使用 GCache 缓存 ===")
	var value3 string
	found, err = gcacheCacher.Get(ctx, "key3", &value3, fallback, &cacher.CacheOptions{TTL: time.Minute})
	if err != nil {
		panic(err)
	}
	if found {
		fmt.Printf("获取到值: %s\n", value3)
	}

	// 批量获取
	fmt.Println("\n=== 批量获取 ===")
	result := make(map[string]string)
	err = redisCacher.MGet(ctx, []string{"key1", "key2", "key3", "key4"}, &result, batchFallback, &cacher.CacheOptions{TTL: time.Minute})
	if err != nil {
		panic(err)
	}
	for k, v := range result {
		fmt.Printf("%s: %s\n", k, v)
	}
}
