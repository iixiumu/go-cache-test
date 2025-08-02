package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	
	cache "go-cache"
	redisStore "go-cache/stores/redis"
	ristrettoStore "go-cache/stores/ristretto"
	gcacheStore "go-cache/stores/gcache"
)

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func main() {
	ctx := context.Background()
	
	// Example 1: Using Redis Store
	fmt.Println("=== Redis Store Example ===")
	if err := redisExample(ctx); err != nil {
		log.Printf("Redis example error: %v", err)
	}
	
	// Example 2: Using Ristretto Store  
	fmt.Println("\n=== Ristretto Store Example ===")
	if err := ristrettoExample(ctx); err != nil {
		log.Printf("Ristretto example error: %v", err)
	}
	
	// Example 3: Using GCache Store
	fmt.Println("\n=== GCache Store Example ===")
	if err := gcacheExample(ctx); err != nil {
		log.Printf("GCache example error: %v", err)
	}
}

func redisExample(ctx context.Context) error {
	// Setup Redis using miniredis for demo
	s, err := miniredis.Run()
	if err != nil {
		return err
	}
	defer s.Close()

	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer client.Close()

	// Create Redis store and cacher
	store := redisStore.NewRedisStore(client)
	cacher := cache.NewCacherWithTTL(store, time.Minute*5)

	// Define fallback functions
	userFallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		fmt.Printf("Fetching user from database for key: %s\n", key)
		// Simulate database lookup
		return User{
			ID:    123,
			Name:  "John Doe", 
			Email: "john@example.com",
		}, true, nil
	}

	batchUserFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		fmt.Printf("Batch fetching users from database for keys: %v\n", keys)
		result := make(map[string]interface{})
		for i, key := range keys {
			result[key] = User{
				ID:    100 + i,
				Name:  fmt.Sprintf("User %d", i+1),
				Email: fmt.Sprintf("user%d@example.com", i+1),
			}
		}
		return result, nil
	}

	// Single get with fallback
	var user User
	found, err := cacher.Get(ctx, "user:123", &user, userFallback, nil)
	if err != nil {
		return err
	}
	fmt.Printf("User found: %v, Data: %+v\n", found, user)

	// Get same user again (should come from cache)
	var user2 User
	found, err = cacher.Get(ctx, "user:123", &user2, userFallback, nil)
	if err != nil {
		return err
	}
	fmt.Printf("User found (cached): %v, Data: %+v\n", found, user2)

	// Batch get with fallback
	keys := []string{"user:1", "user:2", "user:3"}
	userMap := make(map[string]User)
	err = cacher.MGet(ctx, keys, &userMap, batchUserFallback, nil)
	if err != nil {
		return err
	}
	fmt.Printf("Batch users: %+v\n", userMap)

	// Refresh cache
	err = cacher.MRefresh(ctx, keys, &userMap, batchUserFallback, nil)
	if err != nil {
		return err
	}
	fmt.Printf("Refreshed users: %+v\n", userMap)

	return nil
}

func ristrettoExample(ctx context.Context) error {
	// Create Ristretto store and cacher
	store, err := ristrettoStore.NewDefaultRistrettoStore()
	if err != nil {
		return err
	}
	defer store.Close()

	cacher := cache.NewCacherWithTTL(store, time.Minute*10)

	// Simple string cache example
	stringFallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		fmt.Printf("Generating value for key: %s\n", key)
		return fmt.Sprintf("generated_value_for_%s", key), true, nil
	}

	var value string
	found, err := cacher.Get(ctx, "config:setting1", &value, stringFallback, nil)
	if err != nil {
		return err
	}
	fmt.Printf("Config value found: %v, Data: %s\n", found, value)

	// Get same value again (should come from cache)
	var value2 string
	found, err = cacher.Get(ctx, "config:setting1", &value2, stringFallback, nil)
	if err != nil {
		return err
	}
	fmt.Printf("Config value found (cached): %v, Data: %s\n", found, value2)

	return nil
}

func gcacheExample(ctx context.Context) error {
	// Create GCache store and cacher with LRU eviction
	store := gcacheStore.NewLRUGCacheStore(100)
	cacher := cache.NewCacherWithTTL(store, time.Minute*2)

	// Number cache example
	numberFallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		fmt.Printf("Calculating expensive computation for key: %s\n", key)
		// Simulate expensive computation
		return 42 * len(key), true, nil
	}

	batchNumberFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		fmt.Printf("Batch calculating for keys: %v\n", keys)
		result := make(map[string]interface{})
		for _, key := range keys {
			result[key] = 100 + len(key)
		}
		return result, nil
	}

	var result int
	found, err := cacher.Get(ctx, "computation:1", &result, numberFallback, nil)
	if err != nil {
		return err
	}
	fmt.Printf("Computation result found: %v, Data: %d\n", found, result)

	// Batch computation
	keys := []string{"batch:1", "batch:2", "batch:3"}
	resultMap := make(map[string]int)
	err = cacher.MGet(ctx, keys, &resultMap, batchNumberFallback, nil)
	if err != nil {
		return err
	}
	fmt.Printf("Batch computation results: %+v\n", resultMap)

	// Delete some cache entries
	deleted, err := cacher.MDelete(ctx, []string{"batch:1", "batch:2"})
	if err != nil {
		return err
	}
	fmt.Printf("Deleted %d cache entries\n", deleted)

	return nil
}