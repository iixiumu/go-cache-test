package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/alicebob/miniredis/v2"
	redisclient "github.com/redis/go-redis/v9"
	"go-cache/cacher"
	redisstore "go-cache/cacher/store/redis"
	"go-cache/cacher/store/ristretto"
)

// User 示例用户结构
type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func main() {
	ctx := context.Background()

	fmt.Println("=== Go Cache Library 演示 ===")

	// 演示Redis Store
	fmt.Println("\n1. Redis Store 演示:")
	demoRedisStore(ctx)

	// 演示Ristretto Store
	fmt.Println("\n2. Ristretto Store 演示:")
	demoRistrettoStore(ctx)

	// 演示Cacher with fallback
	fmt.Println("\n3. Cacher 回退机制演示:")
	demoCacherFallback(ctx)
}

func demoRedisStore(ctx context.Context) {
	// 启动内存Redis
	mr, err := miniredis.Run()
	if err != nil {
		log.Fatal(err)
	}
	defer mr.Close()

	// 创建Redis客户端
	client := redisclient.NewClient(&redisclient.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	// 创建Redis Store
	store := redisstore.NewStore(client)

	// 测试基本操作
	user := User{ID: 1, Name: "Alice", Age: 30}
	err = store.MSet(ctx, map[string]interface{}{"user:1": user}, 5*time.Second)
	if err != nil {
		log.Fatal(err)
	}

	var retrievedUser User
	found, err := store.Get(ctx, "user:1", &retrievedUser)
	if err != nil {
		log.Fatal(err)
	}

	if found {
		fmt.Printf("Redis Store: 获取用户 %+v\n", retrievedUser)
	} else {
		fmt.Println("Redis Store: 用户未找到")
	}

	// 测试批量操作
	users := map[string]interface{}{
		"user:2": User{ID: 2, Name: "Bob", Age: 25},
		"user:3": User{ID: 3, Name: "Charlie", Age: 35},
	}
	err = store.MSet(ctx, users, 0)
	if err != nil {
		log.Fatal(err)
	}

	keys := []string{"user:1", "user:2", "user:3"}
	userMap := make(map[string]User)
	err = store.MGet(ctx, keys, &userMap)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Redis Store: 批量获取 %d 个用户\n", len(userMap))
}

func demoRistrettoStore(ctx context.Context) {
	// 创建Ristretto Store
	store, err := ristretto.NewStore()
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	// 测试基本操作
	user := User{ID: 1, Name: "Alice", Age: 30}
	err = store.MSet(ctx, map[string]interface{}{"user:1": user}, 5*time.Second)
	if err != nil {
		log.Fatal(err)
	}

	var retrievedUser User
	found, err := store.Get(ctx, "user:1", &retrievedUser)
	if err != nil {
		log.Fatal(err)
	}

	if found {
		fmt.Printf("Ristretto Store: 获取用户 %+v\n", retrievedUser)
	} else {
		fmt.Println("Ristretto Store: 用户未找到")
	}

	// 测试批量操作
	users := map[string]interface{}{
		"user:2": User{ID: 2, Name: "Bob", Age: 25},
		"user:3": User{ID: 3, Name: "Charlie", Age: 35},
	}
	err = store.MSet(ctx, users, 0)
	if err != nil {
		log.Fatal(err)
	}

	keys := []string{"user:1", "user:2", "user:3"}
	userMap := make(map[string]User)
	err = store.MGet(ctx, keys, &userMap)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Ristretto Store: 批量获取 %d 个用户\n", len(userMap))
}

func demoCacherFallback(ctx context.Context) {
	// 创建Ristretto Store
	store, err := ristretto.NewStore()
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	// 创建Cacher
	cache := cacher.NewCacher(store)

	// 模拟数据库
	database := map[string]User{
		"user:1": {ID: 1, Name: "Alice", Age: 30},
		"user:2": {ID: 2, Name: "Bob", Age: 25},
		"user:3": {ID: 3, Name: "Charlie", Age: 35},
	}

	// 单个用户fallback函数
	userFallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		fmt.Printf("Fallback: 从数据库获取 %s\n", key)
		if user, exists := database[key]; exists {
			return user, true, nil
		}
		return nil, false, nil
	}

	// 批量用户fallback函数
	batchUserFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		fmt.Printf("Batch Fallback: 从数据库批量获取 %v\n", keys)
		result := make(map[string]interface{})
		for _, key := range keys {
			if user, exists := database[key]; exists {
				result[key] = user
			}
		}
		return result, nil
	}

	// 第一次获取（缓存未命中，会调用fallback）
	fmt.Println("\n第一次获取（缓存未命中）:")
	var user User
	found, err := cache.Get(ctx, "user:1", &user, userFallback, nil)
	if err != nil {
		log.Fatal(err)
	}
	if found {
		fmt.Printf("获取到用户: %+v\n", user)
	}

	// 第二次获取（缓存命中，不会调用fallback）
	fmt.Println("\n第二次获取（缓存命中）:")
	found, err = cache.Get(ctx, "user:1", &user, userFallback, nil)
	if err != nil {
		log.Fatal(err)
	}
	if found {
		fmt.Printf("获取到用户: %+v\n", user)
	}

	// 批量获取（部分命中）
	fmt.Println("\n批量获取（部分缓存命中）:")
	keys := []string{"user:1", "user:2", "user:3"}
	userMap := make(map[string]User)
	err = cache.MGet(ctx, keys, &userMap, batchUserFallback, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("批量获取到 %d 个用户\n", len(userMap))

	// 刷新缓存
	fmt.Println("\n刷新缓存:")
	err = cache.MRefresh(ctx, []string{"user:1"}, &userMap, batchUserFallback, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("刷新后的用户: %+v\n", userMap["user:1"])

	// 删除缓存
	fmt.Println("\n删除缓存:")
	deletedCount, err := cache.MDelete(ctx, []string{"user:1", "user:2"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("删除了 %d 个缓存项\n", deletedCount)

	// 带TTL的缓存
	fmt.Println("\n带TTL的缓存:")
	opts := &cacher.CacheOptions{TTL: 1 * time.Second}
	found, err = cache.Get(ctx, "user:ttl", &user, userFallback, opts)
	if err != nil {
		log.Fatal(err)
	}
	if found {
		fmt.Printf("设置了TTL的用户: %+v\n", user)
	}

	fmt.Println("\n=== 演示完成 ===")
}