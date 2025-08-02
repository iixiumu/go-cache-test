package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/bluele/gcache"
	"github.com/dgraph-io/ristretto"
	"github.com/go-redis/redis/v8"
)

// ExampleCacher demonstrates how to use the Cacher interface
func ExampleCacher() {
	// 创建内存存储
	store := NewMemoryStore()
	cacher := NewCacher(store)

	ctx := context.Background()

	// 定义回退函数
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		// 模拟从数据库或其他数据源获取数据
		value := fmt.Sprintf("value_for_%s", key)
		return value, true, nil
	}

	// 获取数据
	var result string
	found, err := cacher.Get(ctx, "key1", &result, fallback, nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	if found {
		fmt.Printf("Found value: %s\n", result)
	}

	// 再次获取数据，这次应该从缓存中获取
	found, err = cacher.Get(ctx, "key1", &result, fallback, nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	if found {
		fmt.Printf("Found value from cache: %s\n", result)
	}

	// Output:
	// Found value: value_for_key1
	// Found value from cache: value_for_key1
}

// Example_redisStore demonstrates how to use the Redis Store
func Example_redisStore() {
	// 创建miniredis实例（在实际应用中，您会连接到真实的Redis服务器）
	mr, err := miniredis.Run()
	if err != nil {
		fmt.Printf("Failed to start miniredis: %v\n", err)
		return
	}
	defer mr.Close()

	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// 创建Redis Store
	store := NewRedisStore(client)
	ctx := context.Background()

	// 设置数据
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	err = store.MSet(ctx, items, 0)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// 获取数据
	var result string
	found, err := store.Get(ctx, "key1", &result)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	if found {
		fmt.Printf("Found value: %s\n", result)
	}

	// Output:
	// Found value: value1
}

// Example_ristrettoStore demonstrates how to use the Ristretto Store
func Example_ristrettoStore() {
	// 创建Ristretto缓存
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1000,
		MaxCost:     1000,
		BufferItems: 64,
	})
	if err != nil {
		fmt.Printf("Failed to create Ristretto cache: %v\n", err)
		return
	}

	// 创建Ristretto Store
	store := NewRistrettoStore(cache)
	ctx := context.Background()

	// 设置数据
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	err = store.MSet(ctx, items, 0)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// 等待缓存操作完成
	cache.Wait()

	// 获取数据
	var result string
	found, err := store.Get(ctx, "key1", &result)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	if found {
		fmt.Printf("Found value: %s\n", result)
	}

	// Output:
	// Found value: value1
}

// Example_gcacheStore demonstrates how to use the GCache Store
func Example_gcacheStore() {
	// 创建GCache缓存
	cache := gcache.New(1000).Build()

	// 创建GCache Store
	store := NewGCacheStore(cache)
	ctx := context.Background()

	// 设置数据
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	err := store.MSet(ctx, items, 0)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// 获取数据
	var result string
	found, err := store.Get(ctx, "key1", &result)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	if found {
		fmt.Printf("Found value: %s\n", result)
	}

	// Output:
	// Found value: value1
}

// Example_cacherWithTTL demonstrates how to use the Cacher interface with TTL
func Example_cacherWithTTL() {
	// 创建内存存储
	store := NewMemoryStore()
	cacher := NewCacher(store)

	ctx := context.Background()

	// 定义回退函数
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		// 模拟从数据库或其他数据源获取数据
		value := fmt.Sprintf("value_for_%s", key)
		return value, true, nil
	}

	// 使用TTL选项
	opts := &CacheOptions{
		TTL: 100 * time.Millisecond,
	}

	// 获取数据
	var result string
	found, err := cacher.Get(ctx, "key1", &result, fallback, opts)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	if found {
		fmt.Printf("Found value: %s\n", result)
	}

	// 等待TTL过期
	time.Sleep(150 * time.Millisecond)

	// 再次获取数据，这次应该从回退函数获取，因为缓存已过期
	found, err = cacher.Get(ctx, "key1", &result, fallback, opts)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	if found {
		fmt.Printf("Found value after TTL expired: %s\n", result)
	}

	// Output:
	// Found value: value_for_key1
	// Found value after TTL expired: value_for_key1
}
