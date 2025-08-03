package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go-cache/cacher"
	"go-cache/cacher/store/ristretto"
)

func main() {
	// Create a Ristretto store
	store, err := ristretto.NewRistrettoStore()
	if err != nil {
		log.Fatal(err)
	}

	// Create a cacher with the store
	cache := cacher.NewCacher(store)

	// Example 1: Simple Get with fallback
	fmt.Println("Example 1: Simple Get with fallback")
	ctx := context.Background()

	// Define a fallback function to get data when not in cache
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		// Simulate fetching data from a database or API
		fmt.Printf("Fetching data for key: %s\n", key)
		return "Hello, " + key + "!", true, nil
	}

	// Get value from cache (will use fallback since it's not cached yet)
	var result string
	found, err := cache.Get(ctx, "world", &result, fallback, &cacher.CacheOptions{TTL: 5 * time.Second})
	if err != nil {
		log.Fatal(err)
	}
	if found {
		fmt.Printf("Result: %s\n", result)
	}

	// Get value again (will come from cache this time)
	found, err = cache.Get(ctx, "world", &result, fallback, nil)
	if err != nil {
		log.Fatal(err)
	}
	if found {
		fmt.Printf("Result from cache: %s\n", result)
	}

	// Example 2: Batch Get with fallback
	fmt.Println("\nExample 2: Batch Get with fallback")
	
	// Define a batch fallback function
	batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		// Simulate fetching multiple data items
		fmt.Printf("Fetching data for keys: %v\n", keys)
		result := make(map[string]interface{})
		for _, key := range keys {
			result[key] = "Data for " + key
		}
		return result, nil
	}

	// Get multiple values
	keys := []string{"key1", "key2", "key3"}
	resultMap := make(map[string]string)
	err = cache.MGet(ctx, keys, &resultMap, batchFallback, &cacher.CacheOptions{TTL: 5 * time.Second})
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("Batch result: %v\n", resultMap)

	// Example 3: Delete keys
	fmt.Println("\nExample 3: Delete keys")
	deleted, err := cache.MDelete(ctx, []string{"world", "key1"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Deleted %d keys\n", deleted)

	// Example 4: Refresh cache
	fmt.Println("\nExample 4: Refresh cache")
	
	// Define a refresh fallback function
	refreshFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		// Simulate refreshing data
		fmt.Printf("Refreshing data for keys: %v\n", keys)
		result := make(map[string]interface{})
		for _, key := range keys {
			result[key] = "Refreshed data for " + key
		}
		return result, nil
	}

	// Refresh values in cache
	refreshKeys := []string{"key2", "key3"}
	refreshResultMap := make(map[string]string)
	err = cache.MRefresh(ctx, refreshKeys, &refreshResultMap, refreshFallback, &cacher.CacheOptions{TTL: 10 * time.Second})
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("Refresh result: %v\n", refreshResultMap)
}