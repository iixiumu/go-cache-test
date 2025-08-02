package main

import (
	"context"
	"fmt"
	"time"

	"github.com/example/go-cache/cache"
)

func main() {
	// 创建内存存储
	store := cache.NewMemoryStore()
	cacher := cache.NewCacher(store)

	ctx := context.Background()

	// 定义回退函数
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		// 模拟从数据库或其他数据源获取数据
		fmt.Printf("Fetching data from source for key: %s\n", key)
		value := fmt.Sprintf("value_for_%s", key)
		return value, true, nil
	}

	// 获取数据
	fmt.Println("=== First access (should fetch from source) ===")
	var result string
	found, err := cacher.Get(ctx, "key1", &result, fallback, nil)
	if err != nil {
		panic(err)
	}
	if found {
		fmt.Printf("Found value: %s\n", result)
	}

	// 再次获取数据，这次应该从缓存中获取
	fmt.Println("\n=== Second access (should fetch from cache) ===")
	found, err = cacher.Get(ctx, "key1", &result, fallback, nil)
	if err != nil {
		panic(err)
	}
	if found {
		fmt.Printf("Found value: %s\n", result)
	}

	// 使用TTL选项
	fmt.Println("\n=== Access with TTL ===")
	opts := &cache.CacheOptions{
		TTL: 1 * time.Second,
	}
	var result2 string
	found, err = cacher.Get(ctx, "key2", &result2, fallback, opts)
	if err != nil {
		panic(err)
	}
	if found {
		fmt.Printf("Found value with TTL: %s\n", result2)
	}

	// 等待TTL过期
	fmt.Println("Waiting for TTL to expire...")
	time.Sleep(1100 * time.Millisecond)

	// 再次获取数据，这次应该从回退函数获取，因为缓存已过期
	fmt.Println("\n=== Access after TTL expired (should fetch from source again) ===")
	found, err = cacher.Get(ctx, "key2", &result2, fallback, opts)
	if err != nil {
		panic(err)
	}
	if found {
		fmt.Printf("Found value after TTL expired: %s\n", result2)
	}

	// 批量操作示例
	fmt.Println("\n=== Batch operations ===")
	// 批量设置
	items := map[string]interface{}{
		"batch_key1": "batch_value1",
		"batch_key2": "batch_value2",
		"batch_key3": "batch_value3",
	}
	err = store.MSet(ctx, items, 0)
	if err != nil {
		panic(err)
	}

	// 批量获取
	keys := []string{"batch_key1", "batch_key2", "batch_key3", "batch_key4"}
	batchResult := make(map[string]string)
	err = cacher.MGet(ctx, keys, &batchResult, nil, nil)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Batch get result: %+v\n", batchResult)

	// 批量删除
	deleted, err := cacher.MDelete(ctx, []string{"batch_key1", "batch_key2"})
	if err != nil {
		panic(err)
	}
	fmt.Printf("Deleted %d keys\n", deleted)
}
