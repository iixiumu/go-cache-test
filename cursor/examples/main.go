package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go-cache"
)

func main() {
	// 创建模拟存储
	store := cache.NewMockStore()
	cacher := cache.NewCacher(store)
	ctx := context.Background()

	fmt.Println("=== Go 高级缓存库示例 ===")

	// 示例1: 基本缓存操作
	fmt.Println("\n1. 基本缓存操作")

	// 设置一些初始数据
	store.(*cache.MockStore).SetData("user:1", map[string]interface{}{
		"id":   1,
		"name": "张三",
		"age":  25,
	})

	// 获取缓存数据
	var user map[string]interface{}
	found, err := cacher.Get(ctx, "user:1", &user, nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	if found {
		fmt.Printf("获取到用户: %v\n", user)
	}

	// 示例2: 回退机制
	fmt.Println("\n2. 回退机制")

	// 定义回退函数
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		if key == "user:2" {
			return map[string]interface{}{
				"id":   2,
				"name": "李四",
				"age":  30,
			}, true, nil
		}
		return nil, false, nil
	}

	// 获取不存在的缓存项，使用回退函数
	var user2 map[string]interface{}
	found, err = cacher.Get(ctx, "user:2", &user2, fallback, &cache.CacheOptions{TTL: time.Hour})
	if err != nil {
		log.Fatal(err)
	}
	if found {
		fmt.Printf("通过回退获取到用户: %v\n", user2)
	}

	// 示例3: 批量操作
	fmt.Println("\n3. 批量操作")

	// 设置更多数据
	store.(*cache.MockStore).SetData("user:3", map[string]interface{}{
		"id":   3,
		"name": "王五",
		"age":  28,
	})

	// 批量回退函数
	batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		result := make(map[string]interface{})
		for _, key := range keys {
			if key == "user:4" {
				result[key] = map[string]interface{}{
					"id":   4,
					"name": "赵六",
					"age":  35,
				}
			}
		}
		return result, nil
	}

	// 批量获取
	users := make(map[string]map[string]interface{})
	err = cacher.MGet(ctx, []string{"user:1", "user:2", "user:3", "user:4"}, &users, batchFallback, &cache.CacheOptions{TTL: time.Hour})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("批量获取用户: %v\n", users)

	// 示例4: 强制刷新
	fmt.Println("\n4. 强制刷新")

	// 强制刷新缓存
	err = cacher.MRefresh(ctx, []string{"user:1", "user:2"}, &users, batchFallback, &cache.CacheOptions{TTL: time.Hour})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("刷新后的用户: %v\n", users)

	// 示例5: 删除缓存
	fmt.Println("\n5. 删除缓存")

	deleted, err := cacher.MDelete(ctx, []string{"user:1", "user:2"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("删除了 %d 个缓存项\n", deleted)

	// 验证删除结果
	found, err = cacher.Get(ctx, "user:1", &user, nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	if !found {
		fmt.Println("user:1 已被成功删除")
	}

	fmt.Println("\n=== 示例完成 ===")
}
