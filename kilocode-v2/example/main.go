package main

import (
	"context"
	"fmt"
	"time"

	"go-cache/cacher"
	"go-cache/cacher/store/ristretto"
)

func main() {
	// 创建Ristretto存储
	store, err := ristretto.NewRistrettoStore()
	if err != nil {
		panic(err)
	}

	// 创建Cacher实例
	cache := cacher.NewCacher(store)

	// 示例1: 简单的Get操作
	fmt.Println("=== 示例1: 简单的Get操作 ===")
	ctx := context.Background()

	// 定义回退函数
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		fmt.Printf("执行回退函数获取键 %s 的值\n", key)
		// 模拟从数据库或其他数据源获取数据
		return "从数据源获取的值: " + key, true, nil
	}

	// 获取数据
	var result string
	found, err := cache.Get(ctx, "example_key", &result, fallback, nil)
	if err != nil {
		panic(err)
	}

	if found {
		fmt.Printf("获取到值: %s\n", result)
	} else {
		fmt.Println("未找到值")
	}

	// 再次获取同样的键，这次应该从缓存中获取
	fmt.Println("\n再次获取同样的键:")
	found, err = cache.Get(ctx, "example_key", &result, fallback, nil)
	if err != nil {
		panic(err)
	}

	if found {
		fmt.Printf("从缓存获取到值: %s\n", result)
	} else {
		fmt.Println("未找到值")
	}

	// 示例2: 批量获取操作
	fmt.Println("\n=== 示例2: 批量获取操作 ===")
	keys := []string{"key1", "key2", "key3"}

	// 批量回退函数
	batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		fmt.Printf("执行批量回退函数获取键 %v 的值\n", keys)
		result := make(map[string]interface{})
		for _, key := range keys {
			result[key] = "批量获取的值: " + key
		}
		return result, nil
	}

	// 批量获取数据
	batchResult := make(map[string]string)
	err = cache.MGet(ctx, keys, &batchResult, batchFallback, nil)
	if err != nil {
		panic(err)
	}

	fmt.Printf("批量获取结果: %+v\n", batchResult)

	// 示例3: 带TTL的缓存操作
	fmt.Println("\n=== 示例3: 带TTL的缓存操作 ===")
	ttlOptions := &cacher.CacheOptions{
		TTL: 2 * time.Second,
	}

	// 获取带TTL的数据
	found, err = cache.Get(ctx, "ttl_key", &result, func(ctx context.Context, key string) (interface{}, bool, error) {
		fmt.Printf("执行带TTL的回退函数获取键 %s 的值\n", key)
		return "带TTL的值: " + key, true, nil
	}, ttlOptions)
	if err != nil {
		panic(err)
	}

	if found {
		fmt.Printf("获取到带TTL的值: %s\n", result)
	}

	// 等待TTL过期
	fmt.Println("等待3秒...")
	time.Sleep(3 * time.Second)

	// 再次获取，应该会重新执行回退函数
	fmt.Println("TTL过期后再次获取:")
	found, err = cache.Get(ctx, "ttl_key", &result, func(ctx context.Context, key string) (interface{}, bool, error) {
		fmt.Printf("TTL过期后执行回退函数获取键 %s 的值\n", key)
		return "TTL过期后重新获取的值: " + key, true, nil
	}, nil)
	if err != nil {
		panic(err)
	}

	if found {
		fmt.Printf("重新获取到值: %s\n", result)
	}

	fmt.Println("\n所有示例完成!")
}
