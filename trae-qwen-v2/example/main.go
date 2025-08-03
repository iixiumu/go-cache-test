package main

import (
	"context"
	"fmt"
	"log"
	"time"

	ristrettoV2 "github.com/dgraph-io/ristretto/v2"
	"go-cache/cacher"
	storeRistretto "go-cache/cacher/store/ristretto"
)

func main() {
	// 创建Ristretto缓存实例
	cache, err := ristrettoV2.NewCache(&ristrettoV2.Config[string, interface{}] {
		NumCounters: 1000,
		MaxCost:     100,
		BufferItems: 64,
	})
	if err != nil {
		log.Fatalf("Failed to create Ristretto cache: %v", err)
	}
	defer cache.Close()

	// 创建RistrettoStore实例
	store := storeRistretto.NewRistrettoStore(cache)

	// 创建Cacher实例
	cacheImpl := cacher.NewCacherImpl(store)

	// 定义回退函数
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		fmt.Printf("Fetching data for key: %s\n", key)
		// 模拟从数据库或其他数据源获取数据
		data := fmt.Sprintf("Data for %s", key)
		return data, true, nil
	}

	// 定义批量回退函数
	batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		fmt.Printf("Fetching data for keys: %v\n", keys)
		// 模拟从数据库或其他数据源批量获取数据
		result := make(map[string]interface{})
		for _, key := range keys {
			result[key] = fmt.Sprintf("Data for %s", key)
		}
		return result, nil
	}

	// 使用Get方法获取单个值
	var value string
	found, err := cacheImpl.Get(context.Background(), "key1", &value, fallback, nil)
	if err != nil {
		log.Fatalf("Get failed: %v", err)
	}
	if found {
		fmt.Printf("Got value: %s\n", value)
	}

	// 再次获取相同的值，应该从缓存中获取
	var value2 string
	found, err = cacheImpl.Get(context.Background(), "key1", &value2, fallback, nil)
	if err != nil {
		log.Fatalf("Get failed: %v", err)
	}
	if found {
		fmt.Printf("Got value from cache: %s\n", value2)
	}

	// 使用MGet方法批量获取值
	keys := []string{"key2", "key3", "key4"}
	result := make(map[string]string)
	if err := cacheImpl.MGet(context.Background(), keys, &result, batchFallback, nil); err != nil {
		log.Fatalf("MGet failed: %v", err)
	}
	fmt.Printf("MGet result: %v\n", result)

	// 使用MDelete方法删除值
	deleted, err := cacheImpl.MDelete(context.Background(), []string{"key1"})
	if err != nil {
		log.Fatalf("MDelete failed: %v", err)
	}
	fmt.Printf("Deleted %d keys\n", deleted)

	// 使用MRefresh方法刷新值
	refreshResult := make(map[string]string)
	if err := cacheImpl.MRefresh(context.Background(), []string{"key2"}, &refreshResult, batchFallback, nil); err != nil {
		log.Fatalf("MRefresh failed: %v", err)
	}
	fmt.Printf("Refresh result: %v\n", refreshResult)

	// 使用TTL选项
	opts := &cacher.CacheOptions{
		TTL: 1 * time.Second,
	}

	var value3 string
	found, err = cacheImpl.Get(context.Background(), "key5", &value3, fallback, opts)
	if err != nil {
		log.Fatalf("Get failed: %v", err)
	}
	if found {
		fmt.Printf("Got value with TTL: %s\n", value3)
	}

	// 等待TTL过期
	time.Sleep(1 * time.Second)

	// 再次获取值，应该从回退函数获取
	var value4 string
	found, err = cacheImpl.Get(context.Background(), "key5", &value4, fallback, opts)
	if err != nil {
		log.Fatalf("Get failed: %v", err)
	}
	if found {
		fmt.Printf("Got value after TTL expired: %s\n", value4)
	}
}