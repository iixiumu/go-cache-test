package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/redis/go-redis/v9"
	"github.com/go-cache"
)

// User 示例用户结构体
type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// Product 示例商品结构体
type Product struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

// 模拟数据库
var mockDB = map[string]interface{}{
	"user:1": &User{ID: 1, Name: "Alice", Age: 30},
	"user:2": &User{ID: 2, Name: "Bob", Age: 25},
	"product:1": &Product{ID: 1, Name: "Laptop", Price: 999.99},
	"product:2": &Product{ID: 2, Name: "Phone", Price: 599.99},
}

func main() {
	fmt.Println("=== Go Cache 示例 ===")

	// 示例1: 使用Ristretto内存缓存
	fmt.Println("\n1. 使用Ristretto内存缓存")
	exampleWithRistretto()

	// 示例2: 使用Redis缓存
	fmt.Println("\n2. 使用Redis缓存")
	exampleWithRedis()

	// 示例3: 批量操作
	fmt.Println("\n3. 批量操作")
	exampleBatchOperations()

	// 示例4: 缓存刷新
	fmt.Println("\n4. 缓存刷新")
	exampleCacheRefresh()

	// 示例5: 并发测试
	fmt.Println("\n5. 并发测试")
	exampleConcurrentAccess()
}

func exampleWithRistretto() {
	// 创建Ristretto缓存存储
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,
		MaxCost:     1 << 30,
		BufferItems: 64,
	})
	if err != nil {
		log.Fatal(err)
	}

	store := go_cache.NewRistrettoStore(cache)
	cacher := go_cache.NewCacher(store)
	ctx := context.Background()

	// 获取单个用户
	var user User
	found, err := cacher.Get(ctx, "user:1", &user, getUserFromDB, nil)
	if err != nil {
		log.Fatal(err)
	}
	if found {
		fmt.Printf("从缓存获取用户: %+v\n", user)
	}

	// 再次获取，应该命中缓存
	found, err = cacher.Get(ctx, "user:1", &user, nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	if found {
		fmt.Printf("缓存命中: %+v\n", user)
	}
}

func exampleWithRedis() {
	// 创建Redis客户端
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	// 创建Redis存储
	store := cache.NewRedisStore(rdb)
	cacher := cache.NewCacher(store)
	ctx := context.Background()

	// 获取商品
	var product Product
	found, err := cacher.Get(ctx, "product:1", &product, getProductFromDB, nil)
	if err != nil {
		log.Printf("Redis连接失败，跳过Redis示例: %v", err)
		return
	}
	if found {
		fmt.Printf("从Redis获取商品: %+v\n", product)
	}
}

func exampleBatchOperations() {
	// 创建Ristretto缓存
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,
		MaxCost:     1 << 30,
		BufferItems: 64,
	})
	if err != nil {
		log.Fatal(err)
	}

	store := cache.NewRistrettoStore(cache)
	cacher := cache.NewCacher(store)
	ctx := context.Background()

	// 批量获取用户
	keys := []string{"user:1", "user:2", "user:3"}
	result := make(map[string]User)

	batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		fallbackData := make(map[string]interface{})
		for _, key := range keys {
			if value, exists := mockDB[key]; exists {
				fallbackData[key] = value
			}
		}
		return fallbackData, nil
	}

	err = cacher.MGet(ctx, keys, &result, batchFallback, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("批量获取结果: %+v\n", result)
}

func exampleCacheRefresh() {
	// 创建Ristretto缓存
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,
		MaxCost:     1 << 30,
		BufferItems: 64,
	})
	if err != nil {
		log.Fatal(err)
	}

	store := cache.NewRistrettoStore(cache)
	cacher := cache.NewCacher(store)
	ctx := context.Background()

	// 先设置一些旧数据
	store.MSet(ctx, map[string]interface{}{
		"user:1": &User{ID: 1, Name: "Old Alice", Age: 0},
	}, 0)

	// 刷新缓存
	var user User
	found, err := cacher.Get(ctx, "user:1", &user, getUserFromDB, nil)
	if err != nil {
		log.Fatal(err)
	}
	if found {
		fmt.Printf("刷新前的用户: %+v\n", user)
	}

	// 强制刷新
	refreshFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		fallbackData := make(map[string]interface{})
		for _, key := range keys {
			if value, exists := mockDB[key]; exists {
				fallbackData[key] = value
			}
		}
		return fallbackData, nil
	}

	result := make(map[string]User)
	err = cacher.MRefresh(ctx, []string{"user:1"}, &result, refreshFallback, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("刷新后的用户: %+v\n", result["user:1"])
}

func exampleConcurrentAccess() {
	// 创建Ristretto缓存
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,
		MaxCost:     1 << 30,
		BufferItems: 64,
	})
	if err != nil {
		log.Fatal(err)
	}

	store := go_cache.NewRistrettoStore(cache)
	cacher := go_cache.NewCacherWithLock(store)
	ctx := context.Background()

	var wg sync.WaitGroup
	concurrentRequests := 10

	// 模拟并发访问同一个key
	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			var user User
			found, err := cacher.Get(ctx, "user:1", &user, getUserFromDB, nil)
			if err != nil {
				log.Printf("协程 %d 获取用户失败: %v", id, err)
				return
			}
			if found {
				fmt.Printf("协程 %d 获取用户: %+v\n", id, user)
			}
		}(i)
	}

	wg.Wait()
	fmt.Println("并发测试完成")
}

// getUserFromDB 模拟从数据库获取用户
func getUserFromDB(ctx context.Context, key string) (interface{}, bool, error) {
	// 模拟数据库延迟
	time.Sleep(100 * time.Millisecond)
	
	if value, exists := mockDB[key]; exists {
		return value, true, nil
	}
	return nil, false, nil
}

// getProductFromDB 模拟从数据库获取商品
func getProductFromDB(ctx context.Context, key string) (interface{}, bool, error) {
	// 模拟数据库延迟
	time.Sleep(50 * time.Millisecond)
	
	if value, exists := mockDB[key]; exists {
		return value, true, nil
	}
	return nil, false, nil
}