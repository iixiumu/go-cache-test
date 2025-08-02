# Go Cache Library

一个功能强大的Go语言缓存库，提供了高级缓存抽象和多种存储后端实现。

## 特性

- **高级缓存抽象**: `Cacher`接口提供了带回退机制的缓存操作
- **多种存储后端**: 支持内存、Redis、Ristretto和GCache存储
- **回退机制**: 缓存未命中时自动从数据源获取数据
- **批量操作**: 支持批量获取、设置和删除操作
- **TTL支持**: 支持设置缓存过期时间
- **泛型支持**: 通过反射支持任意数据类型

## 安装

```bash
go get github.com/example/go-cache
```

## 快速开始

### 基本使用

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/example/go-cache/cache"
)

func main() {
    // 创建内存存储
    store := cache.NewMemoryStore()
    cacher := cache.NewCacher(store)
    
    ctx := context.Background()
    
    // 定义回退函数
    fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
        // 模拟从数据库或其他数据源获取数据
        value := fmt.Sprintf("value_for_%s", key)
        return value, true, nil
    }
    
    // 获取数据
    var result string
    found, err := cacher.Get(ctx, "key1", &result, fallback, nil)
    if err != nil {
        panic(err)
    }
    if found {
        fmt.Printf("Found value: %s\n", result)
    }
}
```

### 使用Redis存储

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/go-redis/redis/v8"
    "github.com/example/go-cache/cache"
)

func main() {
    // 创建Redis客户端
    client := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    
    // 创建Redis存储
    store := cache.NewRedisStore(client)
    cacher := cache.NewCacher(store)
    
    ctx := context.Background()
    
    // 设置数据
    items := map[string]interface{}{
        "key1": "value1",
        "key2": "value2",
    }
    err := store.MSet(ctx, items, 0)
    if err != nil {
        panic(err)
    }
    
    // 获取数据
    var result string
    found, err := store.Get(ctx, "key1", &result)
    if err != nil {
        panic(err)
    }
    if found {
        fmt.Printf("Found value: %s\n", result)
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

## 存储后端实现

### 内存存储 (MemoryStore)

基于内存的存储实现，适用于单机应用或测试环境。

### Redis存储 (RedisStore)

基于Redis的存储实现，适用于分布式应用。

### Ristretto存储 (RistrettoStore)

基于Ristretto的高性能内存缓存实现。

### GCache存储 (GCacheStore)

基于GCache的存储实现，支持多种缓存策略。

## 测试

运行测试：

```bash
go test ./cache -v
```

## 许可证

MIT