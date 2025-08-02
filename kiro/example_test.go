package cache

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bluele/gcache"
	"github.com/dgraph-io/ristretto"
	"github.com/go-redis/redis/v8"
)

// User 示例用户结构
type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// ExampleCacher 演示如何使用Cacher
func ExampleCacher() {
	// 创建GCache存储
	gcacheStore := NewGCacheStore(gcache.New(1000).LRU().Build())
	cacher := NewCacher(gcacheStore)

	ctx := context.Background()

	// 定义单个回退函数
	userFallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		// 模拟从数据库获取用户
		if key == "user:1" {
			return User{ID: 1, Name: "Alice", Age: 25}, true, nil
		}
		return nil, false, nil
	}

	// 定义批量回退函数
	batchUserFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		result := make(map[string]interface{})
		for _, key := range keys {
			switch key {
			case "user:1":
				result[key] = User{ID: 1, Name: "Alice", Age: 25}
			case "user:2":
				result[key] = User{ID: 2, Name: "Bob", Age: 30}
			case "user:3":
				result[key] = User{ID: 3, Name: "Charlie", Age: 35}
			}
		}
		return result, nil
	}

	// 缓存选项
	opts := &CacheOptions{TTL: 5 * time.Minute}

	// 1. 单个获取示例
	var user User
	found, err := cacher.Get(ctx, "user:1", &user, userFallback, opts)
	if err != nil {
		log.Printf("Error getting user: %v", err)
		return
	}
	if found {
		fmt.Printf("User found: %+v\n", user)
	}

	// 2. 批量获取示例
	keys := []string{"user:1", "user:2", "user:3"}
	users := make(map[string]User)
	err = cacher.MGet(ctx, keys, &users, batchUserFallback, opts)
	if err != nil {
		log.Printf("Error getting users: %v", err)
		return
	}
	fmt.Printf("Users found: %+v\n", users)

	// 3. 刷新缓存示例
	refreshedUsers := make(map[string]User)
	err = cacher.MRefresh(ctx, keys, &refreshedUsers, batchUserFallback, opts)
	if err != nil {
		log.Printf("Error refreshing users: %v", err)
		return
	}
	fmt.Printf("Refreshed users: %+v\n", refreshedUsers)

	// 4. 删除缓存示例
	deleted, err := cacher.MDelete(ctx, []string{"user:1", "user:2"})
	if err != nil {
		log.Printf("Error deleting users: %v", err)
		return
	}
	fmt.Printf("Deleted %d users from cache\n", deleted)

	// Output:
	// User found: {ID:1 Name:Alice Age:25}
	// Users found: map[user:1:{ID:1 Name:Alice Age:25} user:2:{ID:2 Name:Bob Age:30} user:3:{ID:3 Name:Charlie Age:35}]
	// Refreshed users: map[user:1:{ID:1 Name:Alice Age:25} user:2:{ID:2 Name:Bob Age:30} user:3:{ID:3 Name:Charlie Age:35}]
	// Deleted 2 users from cache
}

// ExampleRedisStore 演示Redis存储的使用
func ExampleRedisStore() {
	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// 创建Redis存储
	store := NewRedisStore(client)
	cacher := NewCacher(store)

	ctx := context.Background()

	// 使用示例
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		return "fallback_value", true, nil
	}

	var result string
	found, err := cacher.Get(ctx, "test_key", &result, fallback, nil)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	if found {
		fmt.Printf("Result: %s\n", result)
	}
}

// ExampleRistrettoStore 演示Ristretto存储的使用
func ExampleRistrettoStore() {
	// 创建Ristretto缓存
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // 10M counters
		MaxCost:     1 << 30, // 1GB
		BufferItems: 64,      // 64 items buffer
	})
	if err != nil {
		log.Printf("Error creating ristretto cache: %v", err)
		return
	}

	// 创建Ristretto存储
	store := NewRistrettoStore(cache)
	cacher := NewCacher(store)

	ctx := context.Background()

	// 使用示例
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		return "ristretto_value", true, nil
	}

	var result string
	found, err := cacher.Get(ctx, "test_key", &result, fallback, nil)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	if found {
		fmt.Printf("Result: %s\n", result)
	}
}