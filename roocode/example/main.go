package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/xiumu/go-cache/cache"
	"github.com/xiumu/go-cache/store"
)

// User 用户结构体示例
type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func main() {
	// 创建Ristretto缓存实例
	rcache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1000,
		MaxCost:     10000,
		BufferItems: 64,
	})
	if err != nil {
		log.Fatal(err)
	}

	// 创建存储实例
	s := store.NewRistrettoStore(rcache)

	// 创建缓存实例
	c := cache.New(s)
	ctx := context.Background()

	// 示例1: 单个获取
	fmt.Println("=== 示例1: 单个获取 ===")
	var user User
	found, err := c.Get(ctx, "user:1", &user, func(ctx context.Context, key string) (interface{}, bool, error) {
		// 模拟从数据库获取用户
		fmt.Printf("从数据库获取用户 %s\n", key)
		return User{ID: 1, Name: "Alice", Age: 25}, true, nil
	}, nil)

	if err != nil {
		log.Fatal(err)
	}

	if found {
		fmt.Printf("获取到用户: %+v\n", user)
	}

	// 再次获取，这次应该从缓存中获取
	fmt.Println("\n再次获取同一用户:")
	found, err = c.Get(ctx, "user:1", &user, func(ctx context.Context, key string) (interface{}, bool, error) {
		// 这次不应该调用这个函数
		fmt.Printf("从数据库获取用户 %s\n", key)
		return User{ID: 1, Name: "Alice", Age: 25}, true, nil
	}, nil)

	if found {
		fmt.Printf("从缓存获取到用户: %+v\n", user)
	}

	// 示例2: 批量获取
	fmt.Println("\n=== 示例2: 批量获取 ===")
	keys := []string{"user:2", "user:3", "user:4"}
	users := make(map[string]User)
	err = c.MGet(ctx, keys, &users, func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		// 模拟从数据库批量获取用户
		fmt.Printf("从数据库批量获取用户: %v\n", keys)
		result := make(map[string]interface{})
		for i, key := range keys {
			id := i + 2
			result[key] = User{ID: id, Name: fmt.Sprintf("User%d", id), Age: 20 + id}
		}
		return result, nil
	}, nil)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("批量获取到用户: %+v\n", users)

	// 示例3: 带TTL的缓存
	fmt.Println("\n=== 示例3: 带TTL的缓存 ===")
	var userWithTTL User
	opts := &cache.CacheOptions{TTL: time.Second * 2}
	found, err = c.Get(ctx, "user:ttl", &userWithTTL, func(ctx context.Context, key string) (interface{}, bool, error) {
		fmt.Printf("从数据库获取用户 %s (带TTL)\n", key)
		return User{ID: 5, Name: "Bob", Age: 30}, true, nil
	}, opts)

	if found {
		fmt.Printf("获取到用户(带TTL): %+v\n", userWithTTL)
	}

	// 等待TTL过期
	fmt.Println("等待3秒...")
	time.Sleep(time.Second * 3)

	// 再次获取，应该会再次调用回退函数
	found, err = c.Get(ctx, "user:ttl", &userWithTTL, func(ctx context.Context, key string) (interface{}, bool, error) {
		fmt.Printf("从数据库获取用户 %s (TTL已过期)\n", key)
		return User{ID: 5, Name: "Bob", Age: 30}, true, nil
	}, opts)

	if found {
		fmt.Printf("TTL过期后重新获取到用户: %+v\n", userWithTTL)
	}

	// 示例4: 批量删除
	fmt.Println("\n=== 示例4: 批量删除 ===")
	count, err := c.MDelete(ctx, []string{"user:1", "user:2"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("删除了 %d 个缓存项\n", count)

	// 验证删除
	found, err = c.Get(ctx, "user:1", &user, nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	if !found {
		fmt.Println("用户 user:1 已被删除")
	}

	// 示例5: 批量刷新
	fmt.Println("\n=== 示例5: 批量刷新 ===")
	keys = []string{"user:1", "user:2"}
	users = make(map[string]User)
	err = c.MRefresh(ctx, keys, &users, func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		// 强制从数据库获取最新数据
		fmt.Printf("强制从数据库刷新用户: %v\n", keys)
		result := make(map[string]interface{})
		for i, key := range keys {
			id := i + 1
			result[key] = User{ID: id, Name: fmt.Sprintf("UpdatedUser%d", id), Age: 25 + id}
		}
		return result, nil
	}, nil)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("刷新后获取到用户: %+v\n", users)
}
