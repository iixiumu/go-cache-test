# Go-Cache

Go-Cache 是一个高级 Go 语言缓存库，提供了业务层缓存抽象。它内部使用 Store 作为存储后端，提供更高级的缓存模式和回退策略。

## 特性

- **多种存储后端支持**：Redis、Ristretto、GCache
- **回退机制**：缓存未命中时自动从数据源获取数据
- **批量操作**：支持批量获取、删除和刷新
- **TTL 支持**：可为缓存项设置过期时间
- **反射支持**：支持多种数据类型，无需使用泛型

## 安装

```bash
go get github.com/yourusername/go-cache
```

## 使用示例

### 基本使用

```go
package main

import (
    "context"
    "fmt"
    "time"

    "go-cache"
    "go-cache/store/redis"
    "github.com/redis/go-redis/v9"
)

func main() {
    // 创建 Redis 存储
    redisClient := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    redisStore := redis.NewRedisStore(redisClient)

    // 创建缓存实例
    cacher := cacher.NewCacher(redisStore)

    // 定义回退函数
    fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
        // 从数据库或其他数据源获取数据
        return fmt.Sprintf("value_for_%s", key), true, nil
    }

    // 获取缓存项
    ctx := context.Background()
    var value string
    found, err := cacher.Get(ctx, "key1", &value, fallback, &cacher.CacheOptions{TTL: time.Minute})
    if err != nil {
        panic(err)
    }
    if found {
        fmt.Printf("获取到值: %s\n", value)
    }
}
```

### 批量操作

```go
// 批量获取
result := make(map[string]string)
err = cacher.MGet(ctx, []string{"key1", "key2", "key3"}, &result, batchFallback, &cacher.CacheOptions{TTL: time.Minute})
if err != nil {
    panic(err)
}

// 批量删除
count, err := cacher.MDelete(ctx, []string{"key1", "key2", "key3"})
if err != nil {
    panic(err)
}
fmt.Printf("删除了 %d 个键\n", count)
```

## 存储后端

### Redis

```go
import "go-cache/store/redis"

redisStore := redis.NewRedisStore(redisClient)
```

### Ristretto

```go
import "go-cache/store/ristretto"

ristrettoStore, err := ristretto.NewRistrettoStore(&ristretto.Config{
    NumCounters: 1000,
    MaxCost:     1000,
    BufferItems: 64,
})
```

### GCache

```go
import "go-cache/store/gcache"

gcacheStore, err := gcache.NewGCacheStore(&gcache.Config{
    Size: 1000,
})
```

## 测试

运行所有测试：

```bash
go test ./...
```

## 许可证

MIT