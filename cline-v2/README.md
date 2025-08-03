# Go Cache Library

一个高性能的Go语言缓存库，提供统一的缓存接口和多种存储后端支持。

## 特性

- **统一接口**: 提供`Cacher`和`Store`两个核心接口，便于扩展和使用
- **多种存储后端**: 支持Redis和Ristretto内存缓存
- **回退机制**: 缓存未命中时自动执行回退函数获取数据
- **批量操作**: 支持批量获取、删除和刷新操作
- **类型安全**: 使用反射机制支持多种数据类型
- **TTL支持**: 支持缓存过期时间设置

## 安装

```bash
go get github.com/your-username/go-cache
```

## 快速开始

### 使用Ristretto内存缓存

```go
package main

import (
    "context"
    "fmt"
    "log"

    "go-cache/cacher"
    "go-cache/cacher/store/ristretto"
)

func main() {
    // 创建Ristretto存储
    store, err := ristretto.NewRistrettoStore()
    if err != nil {
        log.Fatal("创建存储失败:", err)
    }

    // 创建Cacher
    cache := cacher.NewCacher(store)

    ctx := context.Background()

    // 定义回退函数
    fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
        // 模拟从数据库获取数据
        value := fmt.Sprintf("data_for_%s", key)
        return value, true, nil
    }

    // 获取数据
    var result string
    found, err := cache.Get(ctx, "user_123", &result, fallback, nil)
    if err != nil {
        log.Fatal("获取数据失败:", err)
    }

    if found {
        fmt.Printf("获取到数据: %s\n", result)
    }
}
```

### 使用Redis缓存

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "go-cache/cacher"
    "go-cache/cacher/store/redis"

    redisclient "github.com/redis/go-redis/v9"
)

func main() {
    // 创建Redis客户端
    client := redisclient.NewClient(&redisclient.Options{
        Addr: "localhost:6379",
    })

    // 创建Redis存储
    store := redis.NewRedisStore(client)

    // 创建Cacher
    cache := cacher.NewCacher(store)

    ctx := context.Background()

    // 获取数据，设置10秒过期时间
    var result string
    found, err := cache.Get(ctx, "user_456", &result, fallback, &cacher.CacheOptions{
        TTL: 10 * time.Second,
    })
    if err != nil {
        log.Fatal("获取数据失败:", err)
    }

    if found {
        fmt.Printf("获取到Redis数据: %s\n", result)
    }
}
```

## 核心概念

### Store接口

`Store`是底层存储接口，提供基础的键值存储操作：

```go
type Store interface {
    // Get 从存储后端获取单个值
    Get(ctx context.Context, key string, dst interface{}) (bool, error)

    // MGet 批量获取值到map中
    MGet(ctx context.Context, keys []string, dstMap interface{}) error

    // Exists 批量检查键存在性
    Exists(ctx context.Context, keys []string) (map[string]bool, error)

    // MSet 批量设置键值对，支持TTL
    MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error

    // Del 删除指定键
    Del(ctx context.Context, keys ...string) (int64, error)
}
```

### Cacher接口

`Cacher`是高级缓存接口，提供带回退机制的缓存操作：

```go
type Cacher interface {
    // Get 获取单个缓存项，缓存未命中时执行回退函数并缓存结果
    Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error)

    // MGet 批量获取缓存项，支持部分命中和批量回退
    MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error

    // MDelete 批量清除缓存项
    MDelete(ctx context.Context, keys []string) (int64, error)

    // MRefresh 批量强制刷新缓存项
    MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error
}
```

## 高级用法

### 批量操作

```go
// 批量获取
keys := []string{"item_1", "item_2", "item_3"}
result := make(map[string]string)

batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
    values := make(map[string]interface{})
    for _, key := range keys {
        values[key] = getDataFromDatabase(key) // 从数据库获取数据
    }
    return values, nil
}

err := cache.MGet(ctx, keys, &result, batchFallback, nil)

// 批量删除
deleted, err := cache.MDelete(ctx, []string{"item_1", "item_2"})

// 批量刷新
refreshResult := make(map[string]string)
err = cache.MRefresh(ctx, keys, &refreshResult, batchFallback, nil)
```

## 测试

运行所有测试：

```bash
go test ./... -v
```

## 依赖

- [github.com/redis/go-redis/v9](https://github.com/redis/go-redis) - Redis客户端
- [github.com/dgraph-io/ristretto/v2](https://github.com/dgraph-io/ristretto) - 高性能内存缓存
- [github.com/alicebob/miniredis/v2](https://github.com/alicebob/miniredis) - Redis测试工具

## 许可证

MIT
