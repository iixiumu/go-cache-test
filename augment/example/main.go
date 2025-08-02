package main

import (
	"context"
	"fmt"
	"log"
	"time"

	cache "go-cache"
	"go-cache/cacher"
	gcachestore "go-cache/store/gcache"
	redisstore "go-cache/store/redis"
	ristrettostore "go-cache/store/ristretto"

	"github.com/redis/go-redis/v9"
)

// User 示例用户结构
type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// UserService 模拟用户服务
type UserService struct {
	users map[int]*User
}

func NewUserService() *UserService {
	return &UserService{
		users: map[int]*User{
			1: {ID: 1, Name: "Alice", Age: 25},
			2: {ID: 2, Name: "Bob", Age: 30},
			3: {ID: 3, Name: "Charlie", Age: 35},
		},
	}
}

func (s *UserService) GetUser(id int) (*User, bool) {
	user, exists := s.users[id]
	if !exists {
		return nil, false
	}
	// 模拟数据库查询延迟
	time.Sleep(100 * time.Millisecond)
	return user, true
}

func (s *UserService) GetUsers(ids []int) map[int]*User {
	result := make(map[int]*User)
	for _, id := range ids {
		if user, exists := s.users[id]; exists {
			result[id] = user
		}
	}
	// 模拟数据库查询延迟
	time.Sleep(200 * time.Millisecond)
	return result
}

func main() {
	ctx := context.Background()
	userService := NewUserService()

	// 演示不同的Store实现
	fmt.Println("=== Go Cache Library Demo ===")

	// 1. 使用Ristretto Store
	fmt.Println("1. Using Ristretto Store:")
	demoWithRistretto(ctx, userService)

	// 2. 使用GCache Store
	fmt.Println("\n2. Using GCache Store:")
	demoWithGCache(ctx, userService)

	// 3. 使用Redis Store (需要Redis服务器)
	fmt.Println("\n3. Using Redis Store:")
	demoWithRedis(ctx, userService)
}

func demoWithRistretto(ctx context.Context, userService *UserService) {
	// 创建Ristretto Store
	store, err := ristrettostore.NewDefaultRistrettoStore()
	if err != nil {
		log.Fatalf("Failed to create Ristretto store: %v", err)
	}
	defer store.Close()

	// 创建Cacher
	c := cacher.NewDefaultCacher(store, 5*time.Minute)

	demoBasicOperations(ctx, c, userService, "Ristretto")
}

func demoWithGCache(ctx context.Context, userService *UserService) {
	// 创建GCache Store
	store := gcachestore.NewDefaultGCacheStore(1000)

	// 创建Cacher
	c := cacher.NewDefaultCacher(store, 5*time.Minute)

	demoBasicOperations(ctx, c, userService, "GCache")
}

func demoWithRedis(ctx context.Context, userService *UserService) {
	// 创建Redis客户端 (需要Redis服务器运行在localhost:6379)
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// 测试连接
	if err := client.Ping(ctx).Err(); err != nil {
		fmt.Printf("Redis not available, skipping Redis demo: %v\n", err)
		return
	}
	defer client.Close()

	// 创建Redis Store
	store := redisstore.NewRedisStore(client)

	// 创建Cacher
	c := cacher.NewDefaultCacher(store, 5*time.Minute)

	demoBasicOperations(ctx, c, userService, "Redis")
}

func demoBasicOperations(ctx context.Context, c cache.Cacher, userService *UserService, storeName string) {
	fmt.Printf("--- %s Cache Demo ---\n", storeName)

	// 单个用户获取演示
	fmt.Println("Single user cache demo:")

	// 定义单个用户回退函数
	userFallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		fmt.Printf("  Fallback: fetching user from database for key: %s\n", key)
		// 从key中解析用户ID (简化示例，实际应用中可能需要更复杂的解析)
		var userID int
		if _, err := fmt.Sscanf(key, "user:%d", &userID); err != nil {
			return nil, false, err
		}

		user, found := userService.GetUser(userID)
		if !found {
			return nil, false, nil
		}
		return user, true, nil
	}

	// 第一次获取 - 会触发回退函数
	var user1 User
	found, err := c.Get(ctx, "user:1", &user1, userFallback, nil)
	if err != nil {
		log.Printf("Error getting user: %v", err)
		return
	}
	if found {
		fmt.Printf("  Got user: %+v\n", user1)
	}

	// 第二次获取 - 从缓存获取
	var user1Again User
	found, err = c.Get(ctx, "user:1", &user1Again, userFallback, nil)
	if err != nil {
		log.Printf("Error getting user: %v", err)
		return
	}
	if found {
		fmt.Printf("  Got user from cache: %+v\n", user1Again)
	}

	// 批量用户获取演示
	fmt.Println("\nBatch user cache demo:")

	// 定义批量用户回退函数
	batchUserFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		fmt.Printf("  Batch fallback: fetching users from database for keys: %v\n", keys)

		var userIDs []int
		for _, key := range keys {
			var userID int
			if _, err := fmt.Sscanf(key, "user:%d", &userID); err == nil {
				userIDs = append(userIDs, userID)
			}
		}

		users := userService.GetUsers(userIDs)
		result := make(map[string]interface{})
		for _, key := range keys {
			var userID int
			if _, err := fmt.Sscanf(key, "user:%d", &userID); err == nil {
				if user, exists := users[userID]; exists {
					result[key] = user
				}
			}
		}
		return result, nil
	}

	// 批量获取用户 (user:1已在缓存中，user:2和user:3需要从数据库获取)
	keys := []string{"user:1", "user:2", "user:3", "user:999"}
	userMap := make(map[string]*User)
	err = c.MGet(ctx, keys, &userMap, batchUserFallback, nil)
	if err != nil {
		log.Printf("Error batch getting users: %v", err)
		return
	}

	fmt.Printf("  Batch result:\n")
	for key, user := range userMap {
		if user != nil {
			fmt.Printf("    %s: %+v\n", key, *user)
		}
	}

	// 缓存刷新演示
	fmt.Println("\nCache refresh demo:")
	refreshKeys := []string{"user:1", "user:2"}
	refreshMap := make(map[string]*User)
	err = c.MRefresh(ctx, refreshKeys, &refreshMap, batchUserFallback, nil)
	if err != nil {
		log.Printf("Error refreshing cache: %v", err)
		return
	}

	fmt.Printf("  Refreshed cache:\n")
	for key, user := range refreshMap {
		if user != nil {
			fmt.Printf("    %s: %+v\n", key, *user)
		}
	}

	// 缓存删除演示
	fmt.Println("\nCache delete demo:")
	deleted, err := c.MDelete(ctx, []string{"user:1", "user:2"})
	if err != nil {
		log.Printf("Error deleting from cache: %v", err)
		return
	}
	fmt.Printf("  Deleted %d keys from cache\n", deleted)

	fmt.Printf("--- End of %s Demo ---\n", storeName)
}
