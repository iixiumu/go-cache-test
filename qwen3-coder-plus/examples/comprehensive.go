package main

import (
	"context"
	"fmt"
	"time"

	"github.com/bluele/gcache"
	"github.com/dgraph-io/ristretto"
	"github.com/go-redis/redis/v8"

	"github.com/xiumu/go-cache/cache"
	"github.com/xiumu/go-cache/store"
)

func main() {
	fmt.Println("Go Cache Library Examples")
	fmt.Println("========================")

	// 1. 内存存储示例
	memoryStoreExample()

	// 2. Redis存储示例
	redisStoreExample()

	// 3. Ristretto存储示例
	ristrettoStoreExample()

	// 4. GCache存储示例
	gcacheStoreExample()
}

// 内存存储示例
func memoryStoreExample() {
	fmt.Println("\n1. Memory Store Example:")
	fmt.Println("------------------------")

	// 创建内存存储
	memStore := store.NewMemoryStore()

	// 创建缓存器
	cacher := cache.New(memStore)

	// 创建上下文
	ctx := context.Background()

	// 定义回退函数
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		fmt.Printf("  从数据源获取数据，键: %s\n", key)
		value := fmt.Sprintf("memory_value_for_%s", key)
		return value, true, nil
	}

	// 单个获取示例
	var value string
	found, err := cacher.Get(ctx, "memory_key1", &value, fallback, &cache.CacheOptions{TTL: time.Minute})
	if err != nil {
		fmt.Printf("  获取数据出错: %v\n", err)
		return
	}

	if found {
		fmt.Printf("  首次获取到数据: %s\n", value)
	} else {
		fmt.Println("  未找到数据")
	}

	// 再次获取同样的键，这次应该直接从缓存中获取
	found, err = cacher.Get(ctx, "memory_key1", &value, fallback, &cache.CacheOptions{TTL: time.Minute})
	if err != nil {
		fmt.Printf("  获取数据出错: %v\n", err)
		return
	}

	if found {
		fmt.Printf("  从缓存获取到数据: %s\n", value)
	} else {
		fmt.Println("  未找到数据")
	}
}

// Redis存储示例
func redisStoreExample() {
	fmt.Println("\n2. Redis Store Example:")
	fmt.Println("-----------------------")

	// 创建Redis客户端（这里使用本地Redis实例）
	// 在实际使用中，您需要配置正确的Redis地址
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// 测试连接
	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		fmt.Println("  无法连接到Redis服务器，跳过Redis示例")
		return
	}

	// 创建Redis存储
	redisStore := store.NewRedisStore(client)

	// 创建缓存器
	cacher := cache.New(redisStore)

	// 创建上下文
	ctx := context.Background()

	// 定义回退函数
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		fmt.Printf("  从数据源获取数据，键: %s\n", key)
		value := fmt.Sprintf("redis_value_for_%s", key)
		return value, true, nil
	}

	// 单个获取示例
	var value string
	found, err := cacher.Get(ctx, "redis_key1", &value, fallback, &cache.CacheOptions{TTL: time.Minute})
	if err != nil {
		fmt.Printf("  获取数据出错: %v\n", err)
		return
	}

	if found {
		fmt.Printf("  首次获取到数据: %s\n", value)
	} else {
		fmt.Println("  未找到数据")
	}

	// 再次获取同样的键，这次应该直接从缓存中获取
	found, err = cacher.Get(ctx, "redis_key1", &value, fallback, &cache.CacheOptions{TTL: time.Minute})
	if err != nil {
		fmt.Printf("  获取数据出错: %v\n", err)
		return
	}

	if found {
		fmt.Printf("  从缓存获取到数据: %s\n", value)
	} else {
		fmt.Println("  未找到数据")
	}

	// 清理测试数据
	client.Del(ctx, "redis_key1")
}

// Ristretto存储示例
func ristrettoStoreExample() {
	fmt.Println("\n3. Ristretto Store Example:")
	fmt.Println("---------------------------")

	// 创建Ristretto缓存
	ristrettoCache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // number of keys to track frequency of (10M).
		MaxCost:     1 << 30, // maximum cost of cache (1GB).
		BufferItems: 64,      // number of keys per Get buffer.
	})
	if err != nil {
		fmt.Printf("  创建Ristretto缓存失败: %v\n", err)
		return
	}

	// 创建Ristretto存储
	ristrettoStore := store.NewRistrettoStore(ristrettoCache)

	// 创建缓存器
	cacher := cache.New(ristrettoStore)

	// 创建上下文
	ctx := context.Background()

	// 定义回退函数
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		fmt.Printf("  从数据源获取数据，键: %s\n", key)
		value := fmt.Sprintf("ristretto_value_for_%s", key)
		return value, true, nil
	}

	// 单个获取示例
	var value string
	found, err := cacher.Get(ctx, "ristretto_key1", &value, fallback, &cache.CacheOptions{TTL: time.Minute})
	if err != nil {
		fmt.Printf("  获取数据出错: %v\n", err)
		return
	}

	if found {
		fmt.Printf("  首次获取到数据: %s\n", value)
	} else {
		fmt.Println("  未找到数据")
	}

	// 再次获取同样的键，这次应该直接从缓存中获取
	found, err = cacher.Get(ctx, "ristretto_key1", &value, fallback, &cache.CacheOptions{TTL: time.Minute})
	if err != nil {
		fmt.Printf("  获取数据出错: %v\n", err)
		return
	}

	if found {
		fmt.Printf("  从缓存获取到数据: %s\n", value)
	} else {
		fmt.Println("  未找到数据")
	}
}

// GCache存储示例
func gcacheStoreExample() {
	fmt.Println("\n4. GCache Store Example:")
	fmt.Println("------------------------")

	// 创建GCache缓存
	gcacheCache := gcache.New(1000).LRU().Build()

	// 创建GCache存储
	gcacheStore := store.NewGCacheStore(gcacheCache)

	// 创建缓存器
	cacher := cache.New(gcacheStore)

	// 创建上下文
	ctx := context.Background()

	// 定义回退函数
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		fmt.Printf("  从数据源获取数据，键: %s\n", key)
		value := fmt.Sprintf("gcache_value_for_%s", key)
		return value, true, nil
	}

	// 单个获取示例
	var value string
	found, err := cacher.Get(ctx, "gcache_key1", &value, fallback, &cache.CacheOptions{TTL: time.Minute})
	if err != nil {
		fmt.Printf("  获取数据出错: %v\n", err)
		return
	}

	if found {
		fmt.Printf("  首次获取到数据: %s\n", value)
	} else {
		fmt.Println("  未找到数据")
	}

	// 再次获取同样的键，这次应该直接从缓存中获取
	found, err = cacher.Get(ctx, "gcache_key1", &value, fallback, &cache.CacheOptions{TTL: time.Minute})
	if err != nil {
		fmt.Printf("  获取数据出错: %v\n", err)
		return
	}

	if found {
		fmt.Printf("  从缓存获取到数据: %s\n", value)
	} else {
		fmt.Println("  未找到数据")
	}
}