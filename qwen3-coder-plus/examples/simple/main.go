package main

import (
	"context"
	"fmt"
	"time"

	"github.com/xiumu/go-cache/cache"
	"github.com/xiumu/go-cache/store"
)

func main() {
	// 创建内存存储
	memStore := store.NewMemoryStore()

	// 创建缓存器
	cacher := cache.New(memStore)

	// 创建上下文
	ctx := context.Background()

	// 定义回退函数
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		// 模拟从数据库或其他数据源获取数据
		fmt.Printf("从数据源获取数据，键: %s\n", key)
		value := fmt.Sprintf("value_for_%s", key)
		return value, true, nil
	}

	// 单个获取示例
	var value string
	found, err := cacher.Get(ctx, "key1", &value, fallback, &cache.CacheOptions{TTL: time.Minute})
	if err != nil {
		fmt.Printf("获取数据出错: %v\n", err)
		return
	}

	if found {
		fmt.Printf("获取到数据: %s\n", value)
	} else {
		fmt.Println("未找到数据")
	}

	// 再次获取同样的键，这次应该直接从缓存中获取
	found, err = cacher.Get(ctx, "key1", &value, fallback, &cache.CacheOptions{TTL: time.Minute})
	if err != nil {
		fmt.Printf("获取数据出错: %v\n", err)
		return
	}

	if found {
		fmt.Printf("从缓存获取到数据: %s\n", value)
	} else {
		fmt.Println("未找到数据")
	}

	// 批量获取示例
	keys := []string{"key2", "key3", "key4"}
	result := make(map[string]string)

	// 定义批量回退函数
	batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		fmt.Printf("从数据源批量获取数据，键: %v\n", keys)
		result := make(map[string]interface{})
		for _, key := range keys {
			result[key] = fmt.Sprintf("value_for_%s", key)
		}
		return result, nil
	}

	err = cacher.MGet(ctx, keys, &result, batchFallback, &cache.CacheOptions{TTL: time.Minute})
	if err != nil {
		fmt.Printf("批量获取数据出错: %v\n", err)
		return
	}

	fmt.Printf("批量获取结果: %v\n", result)

	// 删除示例
	deleted, err := cacher.MDelete(ctx, []string{"key1", "key2"})
	if err != nil {
		fmt.Printf("删除数据出错: %v\n", err)
		return
	}

	fmt.Printf("删除了 %d 个键\n", deleted)
}