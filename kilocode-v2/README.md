# Go-Cache

Go-Cache是一个高级Go语言缓存库，提供了业务层缓存抽象。它内部使用Store作为存储后端，提供更高级的缓存模式和回退策略。

## 特性

- **统一接口**: 提供统一的Cacher接口，支持多种存储后端
- **多种存储后端**: 支持Redis和Ristretto内存缓存
- **回退机制**: 支持单个和批量回退函数，当缓存未命中时自动从数据源获取数据
- **TTL支持**: 支持设置缓存过期时间
- **反射支持**: 支持多种数据类型，无需使用泛型
- **批量操作**: 支持批量获取、删除和刷新操作

## 安装

```bash
go get github.com/yourusername/go-cache
```

## 快速开始

### 基本使用

```go
package main

import (
    "context"
    "fmt"

    "go-cache/cacher"
    "go-cache/cacher/store/ristretto"
)

func main() {
    // 创建Ristretto存储
    store, err := ristretto.NewRistrettoStore()
    if err != nil {
        panic(err)
    }

    // 创建Cacher实例
    cache := cacher.NewCacher(store)

    // 定义回退函数
    fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
        // 从数据库或其他数据源获取数据
        return "从数据源获取的值: " + key, true, nil
    }

    // 获取数据
    var result string
    found, err := cache.Get(context.Background(), "example_key", &result, fallback, nil)
    if err != nil {
        panic(err)
    }

    if found {
        fmt.Printf("获取到值: %s\n", result)
    }
}
```

### 使用Redis存储

```go
package main

import (
    "context"
    "fmt"

    "github.com/redis/go-redis/v9"
    "go-cache/cacher"
    "go-cache/cacher/store/redis"
)

func main() {
    // 创建Redis客户端
    client := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })

    // 创建Redis存储
    store := redis.NewRedisStore(client)

    // 创建Cacher实例
    cache := cacher.NewCacher(store)

    // 使用缓存...
}
```

## API

### Cacher接口

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

### Store接口

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

## 测试

运行所有测试:

```bash
go test ./...
```

## 许可证

MIT