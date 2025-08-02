# Go Cache Library

一个高性能的Go语言缓存库，提供统一的缓存接口和多种存储后端实现。

## 特性

- **统一接口**: 提供`Store`和`Cacher`两层抽象，支持不同的存储后端
- **多种后端**: 支持Redis、Ristretto、GCache等多种存储后端
- **回退机制**: 内置回退函数支持，缓存未命中时自动从数据源获取并缓存
- **批量操作**: 支持批量获取、设置、删除等操作，提高性能
- **TTL支持**: 支持缓存过期时间设置
- **类型安全**: 使用反射实现多种数据类型的序列化/反序列化
- **测试完备**: 提供完整的单元测试覆盖

## 架构设计

```
┌─────────────────┐
│     Cacher      │  <- 业务层缓存抽象，提供回退机制
├─────────────────┤
│     Store       │  <- 底层存储接口，提供基础操作
├─────────────────┤
│ Redis/Ristretto │  <- 具体存储实现
│    /GCache      │
└─────────────────┘
```

### Store 接口

底层存储接口，提供基础的键值存储操作：

- `Get(ctx, key, dst)` - 获取单个值
- `MGet(ctx, keys, dstMap)` - 批量获取值
- `Exists(ctx, keys)` - 检查键存在性
- `MSet(ctx, items, ttl)` - 批量设置值（支持TTL）
- `Del(ctx, keys...)` - 删除键

### Cacher 接口

高级缓存接口，提供带回退机制的缓存操作：

- `Get(ctx, key, dst, fallback, opts)` - 获取缓存项，支持回退
- `MGet(ctx, keys, dstMap, fallback, opts)` - 批量获取，支持部分命中
- `MDelete(ctx, keys)` - 批量删除
- `MRefresh(ctx, keys, dstMap, fallback, opts)` - 强制刷新缓存

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
    "time"
    
    "go-cache/cacher"
    "go-cache/store/ristretto"
)

func main() {
    ctx := context.Background()
    
    // 创建Ristretto存储后端
    store, err := ristretto.NewDefaultRistrettoStore()
    if err != nil {
        panic(err)
    }
    defer store.Close()
    
    // 创建缓存器
    c := cacher.NewDefaultCacher(store, 5*time.Minute)
    
    // 定义回退函数
    fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
        // 从数据库或其他数据源获取数据
        return "value from database", true, nil
    }
    
    // 获取数据（首次会触发回退函数）
    var result string
    found, err := c.Get(ctx, "my-key", &result, fallback, nil)
    if err != nil {
        panic(err)
    }
    if found {
        fmt.Printf("Got: %s\n", result)
    }
}
```

### 使用不同的存储后端

#### Redis

```go
import (
    "github.com/redis/go-redis/v9"
    "go-cache/store/redis"
)

client := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})
store := redis.NewRedisStore(client)
c := cacher.NewDefaultCacher(store, 5*time.Minute)
```

#### Ristretto

```go
import "go-cache/store/ristretto"

store, err := ristretto.NewDefaultRistrettoStore()
if err != nil {
    panic(err)
}
defer store.Close()
c := cacher.NewDefaultCacher(store, 5*time.Minute)
```

#### GCache

```go
import "go-cache/store/gcache"

store := gcache.NewDefaultGCacheStore(1000) // 最大1000个条目
c := cacher.NewDefaultCacher(store, 5*time.Minute)
```

### 批量操作

```go
// 批量回退函数
batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
    result := make(map[string]interface{})
    for _, key := range keys {
        // 从数据源获取数据
        result[key] = "value for " + key
    }
    return result, nil
}

// 批量获取
keys := []string{"key1", "key2", "key3"}
resultMap := make(map[string]string)
err := c.MGet(ctx, keys, &resultMap, batchFallback, nil)
if err != nil {
    panic(err)
}

// 批量删除
deleted, err := c.MDelete(ctx, keys)
if err != nil {
    panic(err)
}
fmt.Printf("Deleted %d keys\n", deleted)
```

### 自定义TTL

```go
opts := &cache.CacheOptions{
    TTL: 10 * time.Minute,
}

var result string
found, err := c.Get(ctx, "my-key", &result, fallback, opts)
```

## 支持的数据类型

该库使用JSON序列化，支持所有可JSON序列化的Go类型：

- 基本类型：`string`, `int`, `float64`, `bool`等
- 复合类型：`struct`, `slice`, `map`等
- 指针类型
- 自定义类型（实现了JSON序列化接口）

```go
type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
    Age  int    `json:"age"`
}

var user User
found, err := c.Get(ctx, "user:123", &user, userFallback, nil)
```

## 运行示例

```bash
# 运行示例程序
go run example/main.go

# 运行测试
go test ./...

# 运行特定存储的测试
go test ./store/redis
go test ./store/ristretto
go test ./store/gcache
go test ./cacher
```

## 性能特点

- **Redis**: 适合分布式环境，支持持久化，网络延迟较高
- **Ristretto**: 高性能内存缓存，适合单机高并发场景
- **GCache**: 轻量级内存缓存，支持多种淘汰策略

## 贡献

欢迎提交Issue和Pull Request来改进这个库。

## 许可证

MIT License
