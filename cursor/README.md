# Go 高级缓存库

这是一个Go语言的高级缓存库，提供了业务层缓存抽象，支持多种存储后端和回退策略。

## 特性

- **多种存储后端支持**: Redis、Ristretto、GCache
- **回退机制**: 缓存未命中时自动从数据源获取数据
- **批量操作**: 支持批量获取、设置、删除缓存项
- **强制刷新**: 支持强制刷新缓存项
- **TTL支持**: 支持设置缓存过期时间
- **反射支持**: 使用反射实现类型安全的数据转换

## 安装

```bash
go get github.com/your-repo/go-cache
```

## 快速开始

### 使用Redis后端

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/redis/go-redis/v9"
    "github.com/your-repo/go-cache"
)

func main() {
    // 创建Redis客户端
    rdb := redis.NewClient(&redis.Options{
        Addr:     "localhost:6379",
        Password: "",
        DB:       0,
    })

    // 创建Redis存储后端
    store := cache.NewRedisStore(rdb)
    
    // 创建高级缓存
    cacher := cache.NewCacher(store)
    ctx := context.Background()

    // 定义回退函数
    fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
        // 模拟从数据库获取数据
        if key == "user:1" {
            return map[string]interface{}{
                "id":   1,
                "name": "张三",
                "age":  25,
            }, true, nil
        }
        return nil, false, nil
    }

    // 获取缓存项
    var user map[string]interface{}
    found, err := cacher.Get(ctx, "user:1", &user, fallback, &cache.CacheOptions{TTL: time.Hour})
    if err != nil {
        panic(err)
    }
    
    if found {
        fmt.Printf("用户: %v\n", user)
    }
}
```

### 使用Ristretto后端

```go
package main

import (
    "context"
    "time"
    
    "github.com/dgraph-io/ristretto"
    "github.com/your-repo/go-cache"
)

func main() {
    // 创建Ristretto缓存
    ristrettoCache, err := ristretto.NewCache(&ristretto.Config{
        NumCounters: 1e7,     // 计数器数量
        MaxCost:     1 << 30, // 最大成本
        BufferItems: 64,       // 缓冲区大小
    })
    if err != nil {
        panic(err)
    }

    // 创建Ristretto存储后端
    store := cache.NewRistrettoStore(ristrettoCache)
    
    // 创建高级缓存
    cacher := cache.NewCacher(store)
    
    // 使用缓存...
}
```

### 使用GCache后端

```go
package main

import (
    "github.com/bluele/gcache"
    "github.com/your-repo/go-cache"
)

func main() {
    // 创建GCache缓存
    gcacheCache := gcache.New(1000).LRU().Build()

    // 创建GCache存储后端
    store := cache.NewGCacheStore(gcacheCache)
    
    // 创建高级缓存
    cacher := cache.NewCacher(store)
    
    // 使用缓存...
}
```

## API 文档

### Store 接口

底层存储接口，提供基础的键值存储操作：

```go
type Store interface {
    // 获取单个值
    Get(ctx context.Context, key string, dst interface{}) (bool, error)
    
    // 批量获取值
    MGet(ctx context.Context, keys []string, dstMap interface{}) error
    
    // 批量检查键存在性
    Exists(ctx context.Context, keys []string) (map[string]bool, error)
    
    // 批量设置键值对
    MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error
    
    // 删除指定键
    Del(ctx context.Context, keys ...string) (int64, error)
}
```

### Cacher 接口

高级缓存接口，提供带回退机制的缓存操作：

```go
type Cacher interface {
    // 获取单个缓存项，支持回退
    Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error)
    
    // 批量获取缓存项，支持批量回退
    MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error
    
    // 批量删除缓存项
    MDelete(ctx context.Context, keys []string) (int64, error)
    
    // 批量强制刷新缓存项
    MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error
}
```

### 回退函数类型

```go
// 单个回退函数
type FallbackFunc func(ctx context.Context, key string) (interface{}, bool, error)

// 批量回退函数
type BatchFallbackFunc func(ctx context.Context, keys []string) (map[string]interface{}, error)
```

## 高级用法

### 批量操作示例

```go
// 批量获取用户
users := make(map[string]map[string]interface{})
batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
    result := make(map[string]interface{})
    for _, key := range keys {
        // 从数据库批量获取用户
        user := getUserFromDB(key)
        if user != nil {
            result[key] = user
        }
    }
    return result, nil
}

err := cacher.MGet(ctx, []string{"user:1", "user:2", "user:3"}, &users, batchFallback, &cache.CacheOptions{TTL: time.Hour})
```

### 强制刷新缓存

```go
// 强制刷新用户缓存
err := cacher.MRefresh(ctx, []string{"user:1", "user:2"}, &users, batchFallback, &cache.CacheOptions{TTL: time.Hour})
```

### 删除缓存

```go
// 删除多个缓存项
deleted, err := cacher.MDelete(ctx, []string{"user:1", "user:2"})
fmt.Printf("删除了 %d 个缓存项\n", deleted)
```

## 测试

运行测试：

```bash
go test ./...
```

运行特定测试：

```bash
go test -v -run TestCacher_Get
```

## 依赖

- `github.com/redis/go-redis/v9` - Redis客户端
- `github.com/dgraph-io/ristretto` - 高性能内存缓存
- `github.com/bluele/gcache` - 通用缓存库
- `github.com/alicebob/miniredis/v2` - Redis模拟器（用于测试）

## 许可证

MIT License 