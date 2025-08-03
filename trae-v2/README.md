# Go-Cache

一个高级的Go语言缓存库，支持多种存储后端，提供泛型API和回退机制。

## 特性

- 支持多种存储后端（Redis、Ristretto）
- 使用Go泛型提供类型安全的API
- 内置回退机制，缓存未命中时自动从数据源获取
- 批量操作支持（获取、删除、刷新）
- 可自定义键前缀和TTL
- 完整的单元测试覆盖

## 安装

```bash
go get github.com/yourusername/go-cache
```

## 快速开始

### 使用Redis作为存储后端

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go-cache/cacher"
	"go-cache/cacher/store/redis"

	"github.com/redis/go-redis/v9"
)

type User struct {
	ID   int
	Name string
	Age  int
}

func main() {
	ctx := context.Background()

	// 创建Redis客户端
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// 创建Redis存储
	redisStore, err := redis.New(redis.Options{
		Client: redisClient,
	})
	if err != nil {
		log.Fatalf("创建Redis存储失败: %v", err)
	}
	defer redisStore.Close()

	// 创建Redis缓存器
	redisCacher, err := cacher.NewDefaultCacher[User](cacher.DefaultCacherOptions{
		Store:      redisStore,
		Prefix:     "user",
		DefaultTTL: 5 * time.Minute,
	})
	if err != nil {
		log.Fatalf("创建Redis缓存器失败: %v", err)
	}

	// 使用缓存器获取用户
	userID := 1
	user, err := redisCacher.Get(ctx, fmt.Sprintf("%d", userID), cacher.CacheOptions{}, func(ctx context.Context) (User, error) {
		// 这个函数只在缓存未命中时调用
		fmt.Println("从数据库获取用户")
		return getUserFromDB(userID)
	})
	if err != nil {
		log.Printf("获取用户失败: %v", err)
	} else {
		fmt.Printf("获取到用户: %+v\n", user)
	}
}

func getUserFromDB(id int) (User, error) {
	// 在实际应用中，这里会查询数据库
	return User{
		ID:   id,
		Name: fmt.Sprintf("User-%d", id),
		Age:  20 + id,
	}, nil
}
```

### 使用Ristretto作为存储后端

```go
// 创建Ristretto存储
ristrettoStore, err := ristretto.New(ristretto.Options{
	NumCounters: 1e7,     // 预期缓存项数量的10倍
	MaxCost:     1 << 30, // 最大内存使用量: 1GB
	BufferItems: 64,      // 缓冲区大小
})
if err != nil {
	log.Fatalf("创建Ristretto存储失败: %v", err)
}
defer ristrettoStore.Close()

// 创建Ristretto缓存器
ristrettoCacher, err := cacher.NewDefaultCacher[User](cacher.DefaultCacherOptions{
	Store:      ristrettoStore,
	Prefix:     "user",
	DefaultTTL: 5 * time.Minute,
})
if err != nil {
	log.Fatalf("创建Ristretto缓存器失败: %v", err)
}
```

## 批量操作

### 批量获取

```go
userIDs := []int{2, 3, 4}
userKeys := make([]string, len(userIDs))
for i, id := range userIDs {
	userKeys[i] = fmt.Sprintf("%d", id)
}

users, err := cacher.MGet(ctx, userKeys, cacher.CacheOptions{}, func(ctx context.Context, keys []string) (map[string]User, error) {
	// 这个函数只对缓存未命中的键调用
	fmt.Println("批量从数据库获取用户")
	
	// 从数据库获取用户并返回
	result := make(map[string]User)
	// ...
	return result, nil
})
```

### 批量删除

```go
err = cacher.MDelete(ctx, []string{"1", "2", "3"})
```

### 批量刷新

```go
// 创建结果map
resultMap := make(map[string]User)

err = cacher.MRefresh(ctx, []string{"1", "2"}, &resultMap, func(ctx context.Context, keys []string) (map[string]User, error) {
	// 从数据源获取最新数据
	result := make(map[string]User)
	// ...
	return result, nil
})
```

## 自定义TTL

```go
user, err := cacher.Get(ctx, "1", cacher.CacheOptions{
	TTL: 10 * time.Minute, // 覆盖默认TTL
}, fallbackFunc)
```

## 接口

### Cacher接口

```go
type Cacher[T any] interface {
	Get(ctx context.Context, key string, opts CacheOptions, fallback FallbackFunc[T]) (T, error)
	MGet(ctx context.Context, keys []string, opts CacheOptions, batchFallback BatchFallbackFunc[T]) (map[string]T, error)
	MDelete(ctx context.Context, keys []string) error
	MRefresh(ctx context.Context, keys []string, opts CacheOptions, batchFallback BatchFallbackFunc[T]) error
}
```

### Store接口

```go
type Store interface {
	Get(ctx context.Context, key string, value interface{}) (bool, error)
	MGet(ctx context.Context, keys []string, values interface{}) error
	Exists(ctx context.Context, keys []string) (map[string]bool, error)
	MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) (int64, error)
}
```

## 贡献

欢迎提交问题和拉取请求！

## 许可证

MIT