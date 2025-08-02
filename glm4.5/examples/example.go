package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go-cache/cache"
	"go-cache/store"
)

func main() {
	// 示例1: 使用内存存储
	fmt.Println("=== 使用内存存储 ===")
	testStore := store.NewTestStore()
	c := cache.NewCache(testStore)
	ctx := context.Background()

	// 设置回退函数
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		fmt.Printf("执行回退函数获取: %s\n", key)
		return fmt.Sprintf("fallback_%s", key), true, nil
	}

	batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		fmt.Printf("执行批量回退函数获取: %v\n", keys)
		result := make(map[string]interface{})
		for _, key := range keys {
			result[key] = fmt.Sprintf("batch_%s", key)
		}
		return result, nil
	}

	// 测试Get
	var value string
	found, err := c.Get(ctx, "user:1", &value, fallback, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Get user:1: found=%v, value=%s\n", found, value)

	// 测试MGet
	keys := []string{"user:1", "user:2", "user:3"}
	results := make(map[string]string)
	err = c.MGet(ctx, keys, &results, batchFallback, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("MGet results: %v\n", results)

	// 测试MRefresh
	refreshResults := make(map[string]string)
	err = c.MRefresh(ctx, []string{"user:1", "user:4"}, &refreshResults, batchFallback, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("MRefresh results: %v\n", refreshResults)

	// 测试MDelete
	deleted, err := c.MDelete(ctx, []string{"user:1", "user:2"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("MDelete deleted: %d\n", deleted)

	fmt.Println("\n=== 使用Ristretto存储 ===")
	// 示例2: 使用Ristretto存储
	ristrettoStore, err := store.NewRistrettoStore(10000, 1000000)
	if err != nil {
		log.Fatal(err)
	}
	defer ristrettoStore.Close()

	c2 := cache.NewCache(ristrettoStore)

	// 测试带TTL的缓存
	opts := &cache.CacheOptions{TTL: time.Minute}
	found, err = c2.Get(ctx, "cache:with_ttl", &value, fallback, opts)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Get cache:with_ttl: found=%v, value=%s\n", found, value)

	fmt.Println("\n=== 使用GCache存储 ===")
	// 示例3: 使用GCache存储
	gcacheStore := store.NewGCacheStore(1000)
	c3 := cache.NewCache(gcacheStore)

	// 测试批量操作
	batchResults := make(map[string]string)
	err = c3.MGet(ctx, []string{"batch:1", "batch:2"}, &batchResults, batchFallback, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("GCache MGet results: %v\n", batchResults)

	fmt.Println("\n=== 使用Redis存储（需要Redis服务器） ===")
	// 示例4: 使用Redis存储（需要本地Redis服务器）
	redisStore := store.NewRedisStoreWithOptions(store.RedisStoreOptions{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	// 测试Redis连接
	if err := redisStore.Ping(ctx); err == nil {
		c4 := cache.NewCache(redisStore)
		defer redisStore.Close()

		// 测试Redis操作
		found, err = c4.Get(ctx, "redis:key", &value, fallback, nil)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Redis Get redis:key: found=%v, value=%s\n", found, value)
	} else {
		fmt.Println("Redis服务器不可用，跳过Redis测试")
	}

	fmt.Println("\n=== 所有测试完成 ===")
}