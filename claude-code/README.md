# Go Cache Library

一个高性能的Go语言缓存库，提供统一的缓存接口和多种存储后端支持，包含自动回退机制和批量操作功能。

## 特性

- **统一接口**: 提供 `Store` 和 `Cacher` 两层抽象，便于扩展和测试
- **多存储后端**: 支持 Redis、Ristretto、GCache 等多种存储后端
- **自动回退**: 缓存未命中时自动调用回退函数获取数据并缓存
- **批量操作**: 支持批量获取、设置、删除等操作，提高性能
- **泛型支持**: 使用反射实现类型安全，支持任意数据类型
- **TTL支持**: 灵活的过期时间设置
- **完整测试**: 提供全面的单元测试和集成测试

## 安装

```bash
go get go-cache
```

## 快速开始

### 基本用法

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    cache "go-cache"
    "go-cache/stores/gcache"
)

type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

func main() {
    ctx := context.Background()
    
    // 创建存储后端和缓存实例
    store := gcache.NewLRUGCacheStore(1000)
    cacher := cache.NewCacherWithTTL(store, time.Minute*10)
    
    // 定义回退函数
    userFallback := func(ctx context.Context, key string) (interface{}, bool, error) {
        // 从数据库获取用户信息
        return User{ID: 123, Name: "John Doe"}, true, nil
    }
    
    // 获取用户信息（首次会调用回退函数）
    var user User
    found, err := cacher.Get(ctx, "user:123", &user, userFallback, nil)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("User: %+v\n", user)
}
```

## 接口设计

### Store 接口

底层存储接口，提供基础的键值存储操作：

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

高级缓存接口，提供带回退机制的缓存操作：

```go
type Cacher interface {
    Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error)
    MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error
    MDelete(ctx context.Context, keys []string) (int64, error)
    MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error
}
```

## 存储后端

### Redis Store

基于 Redis 的分布式缓存存储：

```go
import (
    "github.com/go-redis/redis/v8"
    redisStore "go-cache/stores/redis"
)

client := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})

store := redisStore.NewRedisStore(client)
cacher := cache.NewCacher(store)
```

### Ristretto Store

基于 Ristretto 的高性能内存缓存：

```go
import ristrettoStore "go-cache/stores/ristretto"

// 使用默认配置
store, err := ristrettoStore.NewDefaultRistrettoStore()
if err != nil {
    panic(err)
}
defer store.Close()

cacher := cache.NewCacher(store)
```

### GCache Store  

基于 GCache 的内存缓存，支持多种淘汰策略：

```go
import gcacheStore "go-cache/stores/gcache"

// LRU 策略
store := gcacheStore.NewLRUGCacheStore(1000)

// 或者指定其他策略
store := gcacheStore.NewGCacheStore(1000, "lfu") // LFU策略
store := gcacheStore.NewGCacheStore(1000, "arc") // ARC策略

cacher := cache.NewCacher(store)
```

## 高级用法

### 批量操作

```go
// 批量回退函数
batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
    result := make(map[string]interface{})
    for _, key := range keys {
        // 从数据库批量获取数据
        result[key] = getUserFromDB(key)
    }
    return result, nil
}

// 批量获取
keys := []string{"user:1", "user:2", "user:3"}
userMap := make(map[string]User)
err := cacher.MGet(ctx, keys, &userMap, batchFallback, nil)

// 批量删除
deleted, err := cacher.MDelete(ctx, keys)

// 批量刷新缓存
err = cacher.MRefresh(ctx, keys, &userMap, batchFallback, nil)
```

### 自定义TTL

```go
// 全局默认TTL
cacher := cache.NewCacherWithTTL(store, time.Hour)

// 单次操作自定义TTL
opts := &cache.CacheOptions{
    TTL: time.Minute * 30,
}

found, err := cacher.Get(ctx, key, &result, fallback, opts)
```

### 复杂数据类型

库支持任意可JSON序列化的数据类型：

```go
type ComplexData struct {
    ID       int                    `json:"id"`
    Name     string                 `json:"name"`
    Tags     []string               `json:"tags"`
    Metadata map[string]interface{} `json:"metadata"`
}

var data ComplexData
found, err := cacher.Get(ctx, "complex:1", &data, fallback, nil)
```

## 测试

运行所有测试：

```bash
go test ./...
```

运行特定存储后端的测试：

```bash
# Redis store tests
go test ./stores/redis

# Ristretto store tests  
go test ./stores/ristretto

# GCache store tests
go test ./stores/gcache

# Cacher tests
go test -run TestDefaultCacher
```

## 性能考虑

1. **批量操作**: 尽量使用 `MGet`、`MSet` 等批量操作来减少网络开销
2. **TTL设置**: 根据数据特性合理设置TTL，避免缓存雪崩
3. **存储选择**: 
   - Redis: 适合分布式场景和持久化需求
   - Ristretto: 适合高并发内存缓存场景
   - GCache: 适合简单的内存缓存需求

## 最佳实践

1. **错误处理**: 回退函数中的错误会导致整个缓存操作失败，请妥善处理
2. **缓存击穿**: 对于热点数据，考虑在回退函数中实现锁机制
3. **监控指标**: 在生产环境中添加缓存命中率等监控指标
4. **内存管理**: 内存缓存要注意设置合适的大小限制

## 示例

完整的使用示例请参考 `examples/main.go` 文件。

## 许可证

MIT License