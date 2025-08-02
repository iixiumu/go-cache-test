# Go Cache Library

一个高性能的Go语言缓存库，提供统一的缓存接口和多种存储后端支持。

## 特性

- **统一接口**: 提供`Store`和`Cacher`两层抽象，支持不同的存储后端
- **多种存储**: 支持Redis、Ristretto、GCache等多种存储后端
- **回退机制**: 内置缓存未命中时的回退函数支持
- **批量操作**: 支持批量获取、设置、删除等操作
- **TTL支持**: 支持缓存过期时间设置
- **类型安全**: 使用反射实现多类型支持，避免泛型复杂性
- **测试完备**: 提供完整的单元测试覆盖

## 安装

```bash
go get github.com/your-repo/go-cache
```

## 快速开始

### 基本使用

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/bluele/gcache"
    cache "github.com/your-repo/go-cache"
)

func main() {
    // 创建存储后端
    store := cache.NewGCacheStore(gcache.New(1000).LRU().Build())
    
    // 创建缓存器
    cacher := cache.NewCacher(store)
    
    ctx := context.Background()
    
    // 定义回退函数
    fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
        // 从数据库或其他数据源获取数据
        return "data from database", true, nil
    }
    
    // 获取数据（如果缓存未命中，会调用回退函数）
    var result string
    found, err := cacher.Get(ctx, "my_key", &result, fallback, &cache.CacheOptions{
        TTL: 5 * time.Minute,
    })
    
    if err != nil {
        panic(err)
    }
    
    if found {
        fmt.Printf("Result: %s\n", result)
    }
}
```

### 批量操作

```go
// 批量获取
keys := []string{"user:1", "user:2", "user:3"}
users := make(map[string]User)

batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
    result := make(map[string]interface{})
    for _, key := range keys {
        // 从数据库批量获取
        user := getUserFromDB(key)
        if user != nil {
            result[key] = user
        }
    }
    return result, nil
}

err := cacher.MGet(ctx, keys, &users, batchFallback, &cache.CacheOptions{
    TTL: 10 * time.Minute,
})
```

## 存储后端

### Redis

```go
import (
    "github.com/go-redis/redis/v8"
    cache "github.com/your-repo/go-cache"
)

client := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})

store := cache.NewRedisStore(client)
cacher := cache.NewCacher(store)
```

### Ristretto

```go
import (
    "github.com/dgraph-io/ristretto"
    cache "github.com/your-repo/go-cache"
)

ristrettoCache, err := ristretto.NewCache(&ristretto.Config{
    NumCounters: 1e7,     // 10M counters
    MaxCost:     1 << 30, // 1GB
    BufferItems: 64,      // 64 items buffer
})
if err != nil {
    panic(err)
}

store := cache.NewRistrettoStore(ristrettoCache)
cacher := cache.NewCacher(store)
```

### GCache

```go
import (
    "github.com/bluele/gcache"
    cache "github.com/your-repo/go-cache"
)

gcacheInstance := gcache.New(1000).LRU().Build()
store := cache.NewGCacheStore(gcacheInstance)
cacher := cache.NewCacher(store)
```

## API 文档

### Store 接口

`Store`接口提供底层存储操作：

```go
type Store interface {
    Get(ctx context.Context, key string, dst interface{}) (bool, error)
    MGet(ctx context.Context, keys []string, dstMap interface{}) error
    Exists(ctx context.Context, keys []string) (map[string]bool, error)
    MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error
    Del(ctx context.Context, keys ...string) (int64, error)
}
```

### Cacher 接口

`Cacher`接口提供高级缓存操作：

```go
type Cacher interface {
    Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error)
    MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error
    MDelete(ctx context.Context, keys []string) (int64, error)
    MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error
}
```

### 回退函数

```go
// 单个回退函数
type FallbackFunc func(ctx context.Context, key string) (interface{}, bool, error)

// 批量回退函数
type BatchFallbackFunc func(ctx context.Context, keys []string) (map[string]interface{}, error)
```

### 缓存选项

```go
type CacheOptions struct {
    TTL time.Duration // 缓存过期时间，0表示永不过期
}
```

## 测试

运行所有测试：

```bash
go test -v
```

运行特定测试：

```bash
go test -v -run TestCacher
go test -v -run TestRedisStore
go test -v -run TestRistrettoStore
go test -v -run TestGCacheStore
```

## 性能考虑

1. **Redis**: 适合分布式环境，支持持久化，但有网络开销
2. **Ristretto**: 高性能内存缓存，适合单机高并发场景
3. **GCache**: 轻量级内存缓存，适合简单场景

## 贡献

欢迎提交Issue和Pull Request来改进这个库。

## 许可证

MIT License