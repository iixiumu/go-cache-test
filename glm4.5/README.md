# Go Cache Library

一个高级的Go语言缓存库，支持多种存储后端和回退策略。

## 特性

- **多层抽象**: `Cacher`接口提供高级缓存功能，`Store`接口提供底层存储操作
- **多种存储后端**: 支持内存、Ristretto、GCache、Redis等多种存储实现
- **回退策略**: 支持单键和批量回退函数，缓存未命中时从数据源获取数据
- **反射支持**: 使用反射实现多数据类型支持，避免泛型复杂性
- **TTL支持**: 支持缓存过期时间设置
- **批量操作**: 支持批量获取、设置、删除和刷新操作

## 安装

```bash
go get go-cache
```

## 快速开始

### 基本使用

```go
package main

import (
    "context"
    "fmt"
    "go-cache/cache"
    "go-cache/store"
)

func main() {
    // 创建存储后端
    store := store.NewTestStore()
    
    // 创建缓存实例
    c := cache.NewCache(store)
    
    ctx := context.Background()
    
    // 定义回退函数
    fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
        // 从数据库或其他数据源获取数据
        return fmt.Sprintf("data_for_%s", key), true, nil
    }
    
    // 获取缓存
    var result string
    found, err := c.Get(ctx, "user:1", &result, fallback, nil)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Found: %v, Value: %s\n", found, result)
}
```

### 批量操作

```go
// 批量获取
keys := []string{"user:1", "user:2", "user:3"}
results := make(map[string]string)

batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
    result := make(map[string]interface{})
    for _, key := range keys {
        result[key] = fmt.Sprintf("data_for_%s", key)
    }
    return result, nil
}

err := c.MGet(ctx, keys, &results, batchFallback, nil)
if err != nil {
    panic(err)
}

fmt.Printf("Results: %v\n", results)

// 批量删除
deleted, err := c.MDelete(ctx, keys)
if err != nil {
    panic(err)
}
fmt.Printf("Deleted %d items\n", deleted)
```

## 存储后端

### 内存存储 (TestStore)

```go
store := store.NewTestStore()
c := cache.NewCache(store)
```

### Ristretto存储

```go
ristrettoStore, err := store.NewRistrettoStore(10000, 1000000)
if err != nil {
    panic(err)
}
defer ristrettoStore.Close()

c := cache.NewCache(ristrettoStore)
```

### GCache存储

```go
// ARC策略（默认）
gcacheStore := store.NewGCacheStore(1000)

// LRU策略
gcacheStore := store.NewGCacheStoreWithLRU(1000)

// LFU策略
gcacheStore := store.NewGCacheStoreWithLFU(1000)

c := cache.NewCache(gcacheStore)
```

### Redis存储

```go
redisStore := store.NewRedisStoreWithOptions(store.RedisStoreOptions{
    Addr:     "localhost:6379",
    Password: "",
    DB:       0,
})
defer redisStore.Close()

c := cache.NewCache(redisStore)
```

## 高级功能

### TTL设置

```go
opts := &cache.CacheOptions{
    TTL: time.Hour, // 1小时后过期
}

found, err := c.Get(ctx, "key", &value, fallback, opts)
```

### 强制刷新

```go
// 强制刷新多个键的缓存
refreshResults := make(map[string]string)
err := c.MRefresh(ctx, []string{"key1", "key2"}, &refreshResults, batchFallback, nil)
```

## 接口文档

### Store接口

```go
type Store interface {
    Get(ctx context.Context, key string, dst interface{}) (bool, error)
    MGet(ctx context.Context, keys []string, dstMap interface{}) error
    Exists(ctx context.Context, keys []string) (map[string]bool, error)
    MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error
    Del(ctx context.Context, keys ...string) (int64, error)
}
```

### Cacher接口

```go
type Cacher interface {
    Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error)
    MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error
    MDelete(ctx context.Context, keys []string) (int64, error)
    MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error
}
```

## 运行测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./cache
go test ./store

# 运行示例
go run example.go
```

## 依赖

- `github.com/go-redis/redis/v8` - Redis客户端
- `github.com/hypermodeinc/ristretto` - 高性能内存缓存
- `github.com/bluele/gcache` - GCache库
- `github.com/alicebob/miniredis` - Redis测试服务器（测试用）

## 性能考虑

- 使用反射实现多类型支持，会有一定的性能开销
- 批量操作比单次操作更高效
- 内存存储（Ristretto、GCache）比Redis存储更快
- 建议根据实际场景选择合适的存储后端

## 许可证

MIT License