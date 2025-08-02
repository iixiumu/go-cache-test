package main

import (
	"context"
	"fmt"
	"time"

	"github.com/bluele/gcache"
	"github.com/xiumu/go-cache/cache"
	"github.com/xiumu/go-cache/store/gcache"
)

func main() {
	// 创建一个GCache实例
	gc := gcache.New(1000).LRU().Build()
	
	// 创建GCache存储
	store := gcache.New(gc)
	
	// 创建Cacher实例
	cacher := cache.New(store)
	
	// 使用示例
	ctx := context.Background()
	
	// 单个获取示例
	var value string
	found, err := cacher.Get(ctx, "key1", &value, func(ctx context.Context, key string) (interface{}, bool, error) {
		// 模拟从数据库或其他数据源获取数据
		return "data_from_source", true, nil
	}, nil)
	
	if err != nil {
		fmt.Printf("Get error: %v\n", err)
		return
	}
	
	if found {
		fmt.Printf("Found value: %s\n", value)
	} else {
		fmt.Println("Value not found")
	}
	
	// 批量获取示例
	keys := []string{"key1", "key2", "key3"}
	resultMap := make(map[string]interface{})
	
	err = cacher.MGet(ctx, keys, &resultMap, func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		// 模拟从数据库或其他数据源批量获取数据
		result := make(map[string]interface{})
		for _, key := range keys {
			result[key] = fmt.Sprintf("data_for_%s", key)
		}
		return result, nil
	}, nil)
	
	if err != nil {
		fmt.Printf("MGet error: %v\n", err)
		return
	}
	
	fmt.Printf("Batch get results: %+v\n", resultMap)
	
	// 批量删除示例
	count, err := cacher.MDelete(ctx, []string{"key1", "key2"})
	if err != nil {
		fmt.Printf("MDelete error: %v\n", err)
		return
	}
	
	fmt.Printf("Deleted %d keys\n", count)
}