package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go-cache/cacher"
	"go-cache/cacher/store/redis"
	"go-cache/cacher/store/ristretto"

	redisclient "github.com/redis/go-redis/v9"
)

func main() {
	// 演示使用Ristretto内存缓存
	fmt.Println("=== 使用Ristretto内存缓存 ===")
	demoRistretto()

	// 演示使用Redis缓存
	fmt.Println("\n=== 使用Redis缓存 ===")
	demoRedis()

	// 演示高级功能
	fmt.Println("\n=== 高级功能演示 ===")
	demoAdvancedFeatures()
}

func demoRistretto() {
	// 创建Ristretto存储
	store, err := ristretto.NewRistrettoStore()
	if err != nil {
		log.Fatal("创建Ristretto存储失败:", err)
	}

	// 创建Cacher
	cache := cacher.NewCacher(store)

	ctx := context.Background()

	// 定义回退函数
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		fmt.Printf("执行回退函数获取键: %s\n", key)
		// 模拟从数据库或其他数据源获取数据
		value := fmt.Sprintf("data_for_%s", key)
		return value, true, nil
	}

	// 获取数据（缓存未命中，执行回退函数）
	var result string
	found, err := cache.Get(ctx, "user_123", &result, fallback, nil)
	if err != nil {
		log.Fatal("获取数据失败:", err)
	}

	if found {
		fmt.Printf("获取到数据: %s\n", result)
	}

	// 再次获取相同数据（缓存命中）
	found, err = cache.Get(ctx, "user_123", &result, fallback, nil)
	if err != nil {
		log.Fatal("获取数据失败:", err)
	}

	if found {
		fmt.Printf("从缓存获取到数据: %s\n", result)
	}
}

func demoRedis() {
	// 创建Redis客户端（这里使用本地Redis，实际使用时需要配置正确的地址）
	client := redisclient.NewClient(&redisclient.Options{
		Addr: "localhost:6379",
	})

	// 测试连接
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		fmt.Println("无法连接到Redis服务器，跳过Redis演示")
		return
	}

	// 创建Redis存储
	store := redis.NewRedisStore(client)

	// 创建Cacher
	cache := cacher.NewCacher(store)

	// 定义回退函数
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		fmt.Printf("执行回退函数获取键: %s\n", key)
		// 模拟从数据库获取数据
		value := fmt.Sprintf("redis_data_for_%s", key)
		return value, true, nil
	}

	// 获取数据
	var result string
	found, err := cache.Get(ctx, "redis_user_456", &result, fallback, &cacher.CacheOptions{
		TTL: 10 * time.Second, // 设置10秒过期时间
	})
	if err != nil {
		log.Fatal("获取数据失败:", err)
	}

	if found {
		fmt.Printf("获取到Redis数据: %s\n", result)
	}
}

func demoAdvancedFeatures() {
	// 创建Ristretto存储
	store, err := ristretto.NewRistrettoStore()
	if err != nil {
		log.Fatal("创建Ristretto存储失败:", err)
	}

	// 创建Cacher
	cache := cacher.NewCacher(store)

	ctx := context.Background()

	// 批量获取数据
	keys := []string{"item_1", "item_2", "item_3", "item_4"}
	result := make(map[string]string)

	// 批量回退函数
	batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		fmt.Printf("执行批量回退函数获取键: %v\n", keys)
		values := make(map[string]interface{})
		for _, key := range keys {
			values[key] = fmt.Sprintf("batch_data_for_%s", key)
		}
		return values, nil
	}

	// 批量获取
	err = cache.MGet(ctx, keys, &result, batchFallback, nil)
	if err != nil {
		log.Fatal("批量获取数据失败:", err)
	}

	fmt.Printf("批量获取结果: %+v\n", result)

	// 批量删除
	deleted, err := cache.MDelete(ctx, []string{"item_1", "item_2"})
	if err != nil {
		log.Fatal("批量删除失败:", err)
	}
	fmt.Printf("删除了 %d 个键\n", deleted)

	// 批量刷新
	refreshKeys := []string{"item_3", "item_4", "item_5"}
	refreshResult := make(map[string]string)

	refreshFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		fmt.Printf("执行刷新回退函数获取键: %v\n", keys)
		values := make(map[string]interface{})
		for _, key := range keys {
			values[key] = fmt.Sprintf("refreshed_data_for_%s", key)
		}
		return values, nil
	}

	err = cache.MRefresh(ctx, refreshKeys, &refreshResult, refreshFallback, nil)
	if err != nil {
		log.Fatal("批量刷新失败:", err)
	}

	fmt.Printf("刷新结果: %+v\n", refreshResult)
}
