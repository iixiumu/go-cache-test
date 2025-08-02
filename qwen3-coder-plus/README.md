# Go Cache Library

一个高性能的Go语言缓存库，提供了统一的缓存接口和多种存储后端实现。

## 特性

- **统一接口**: 提供了`Store`和`Cacher`两个核心接口，简化缓存操作
- **多种存储后端**: 支持内存、Redis、Ristretto和GCache等多种存储后端
- **回退机制**: 支持缓存未命中时的回退函数，自动从数据源加载数据
- **批量操作**: 支持批量获取、设置和删除操作，提高性能
- **TTL支持**: 支持过期时间设置
- **泛型支持**: 使用反射实现，支持任意数据类型

## 安装

```bash
go get github.com/xiumu/go-cache
```

## 快速开始

### 1. 创建内存缓存

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/xiumu/go-cache/cache"
    "github.com/xiumu/go-cache/store"
)

func main() {
    // 创建内存存储
    memStore := store.NewMemoryStore()

    // 创建缓存器
    cacher := cache.New(memStore)

    // 创建上下文
    ctx := context.Background()

    // 定义回退函数
    fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
        // 模拟从数据库获取数据
        value := fmt.Sprintf("value_for_%s", key)
        return value, true, nil
    }

    // 获取数据
    var value string
    found, err := cacher.Get(ctx, "key1", &value, fallback, &cache.CacheOptions{TTL: time.Minute})
    if err != nil {
        fmt.Printf("获取数据出错: %v\n", err)
        return
    }

    if found {
        fmt.Printf("获取到数据: %s\n", value)
    }
}
```

### 2. 使用Redis存储

```go
// 创建Redis客户端
client := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})

// 创建Redis存储
redisStore := store.NewRedisStore(client)

// 创建缓存器
cacher := cache.New(redisStore)
```

### 3. 使用Ristretto存储

```go
// 创建Ristretto缓存
ristrettoCache, err := ristretto.NewCache(&ristretto.Config{
    NumCounters: 1e7,     // number of keys to track frequency of (10M).
    MaxCost:     1 << 30, // maximum cost of cache (1GB).
    BufferItems: 64,      // number of keys per Get buffer.
})
if err != nil {
    // handle error
}

// 创建Ristretto存储
ristrettoStore := store.NewRistrettoStore(ristrettoCache)

// 创建缓存器
cacher := cache.New(ristrettoStore)
```

### 4. 使用GCache存储

```go
// 创建GCache缓存
gcacheCache := gcache.New(1000).LRU().Build()

// 创建GCache存储
gcacheStore := store.NewGCacheStore(gcacheCache)

// 创建缓存器
cacher := cache.New(gcacheStore)
```

## 核心接口

### Store 接口

`Store`是底层存储接口，提供基础的键值存储操作：

- `Get`: 获取单个值
- `MGet`: 批量获取值
- `Exists`: 批量检查键存在性
- `MSet`: 批量设置键值对
- `Del`: 删除指定键

### Cacher 接口

`Cacher`是高级缓存接口，提供带回退机制的缓存操作：

- `Get`: 获取单个缓存项，支持回退函数
- `MGet`: 批量获取缓存项，支持部分命中和批量回退
- `MDelete`: 批量清除缓存项
- `MRefresh`: 批量强制刷新缓存项

## 运行测试

```bash
go test ./...
```

## 许可证

MIT