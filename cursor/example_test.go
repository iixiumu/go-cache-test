package cache

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// Example_usage 展示如何使用缓存库的示例
func Example_usage() {
	// 创建Redis客户端
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	// 创建Redis存储后端
	store := NewRedisStore(rdb)

	// 创建高级缓存
	cacher := NewCacher(store)
	ctx := context.Background()

	// 定义回退函数，模拟从数据库获取数据
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		// 模拟数据库查询
		if key == "user:1" {
			return map[string]interface{}{
				"id":   1,
				"name": "张三",
				"age":  25,
			}, true, nil
		}
		return nil, false, nil
	}

	// 批量回退函数
	batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		result := make(map[string]interface{})
		for _, key := range keys {
			if key == "user:2" {
				result[key] = map[string]interface{}{
					"id":   2,
					"name": "李四",
					"age":  30,
				}
			} else if key == "user:3" {
				result[key] = map[string]interface{}{
					"id":   3,
					"name": "王五",
					"age":  28,
				}
			}
		}
		return result, nil
	}

	// 获取单个缓存项
	var user map[string]interface{}
	found, err := cacher.Get(ctx, "user:1", &user, fallback, &CacheOptions{TTL: time.Hour})
	if err != nil {
		log.Fatal(err)
	}
	if found {
		fmt.Printf("获取到用户: %v\n", user)
	}

	// 批量获取缓存项
	users := make(map[string]map[string]interface{})
	err = cacher.MGet(ctx, []string{"user:1", "user:2", "user:3"}, &users, batchFallback, &CacheOptions{TTL: time.Hour})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("批量获取用户: %v\n", users)

	// 强制刷新缓存
	err = cacher.MRefresh(ctx, []string{"user:1", "user:2"}, &users, batchFallback, &CacheOptions{TTL: time.Hour})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("刷新后的用户: %v\n", users)

	// 删除缓存项
	deleted, err := cacher.MDelete(ctx, []string{"user:1", "user:2"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("删除了 %d 个缓存项\n", deleted)
}

// Example_with_mock_store 使用模拟存储的示例
func Example_with_mock_store() {
	// 创建模拟存储
	store := NewMockStore()
	cacher := NewCacher(store)
	ctx := context.Background()

	// 设置一些初始数据
	store.(*MockStore).data["config:app_name"] = "MyApp"
	store.(*MockStore).data["config:version"] = "1.0.0"

	// 定义回退函数
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		if key == "config:debug" {
			return true, true, nil
		}
		return nil, false, nil
	}

	// 获取配置
	var appName string
	found, err := cacher.Get(ctx, "config:app_name", &appName, nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	if found {
		fmt.Printf("应用名称: %s\n", appName)
	}

	// 获取不存在的配置，使用回退函数
	var debug bool
	found, err = cacher.Get(ctx, "config:debug", &debug, fallback, &CacheOptions{TTL: time.Minute * 30})
	if err != nil {
		log.Fatal(err)
	}
	if found {
		fmt.Printf("调试模式: %v\n", debug)
	}

	// 批量获取配置
	configs := make(map[string]interface{})
	err = cacher.MGet(ctx, []string{"config:app_name", "config:version", "config:debug"}, &configs, nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("所有配置: %v\n", configs)
}
