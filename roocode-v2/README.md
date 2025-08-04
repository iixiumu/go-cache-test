# Go-Cache

Go-Cache是一个高性能的Go语言缓存库，提供了统一的缓存接口和多种存储后端实现。

## 特性

- **统一接口**: 提供`Cacher`和`Store`两个接口，便于扩展和替换
- **多存储后端**: 支持Redis和Ristretto两种存储后端
- **回退机制**: 支持缓存未命中时的回退函数
- **类型安全**: 使用反射实现泛型支持，支持多种数据类型
- **批量操作**: 支持批量获取、设置和删除操作
- **TTL支持**: 支持设置缓存项的过期时间

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

    "go-cache/cacher"
    "go-cache/cacher/store/ristretto"
)

func main() {
    // 创建Ristretto存储
    store, err := ristretto.NewRistrettoStore()
    if err != nil {
        panic(err)
    }

    // 创建Cacher
    cache := cacher.NewCacher(store)

    ctx := context.Background()

    // 定义回退函数
    fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
        // 模拟从数据库或其他数据源获取数据
        if key == "user:1" {
            return "John Doe", true, nil
        }
        return nil, false, nil
    }

    // 获取缓存项
    var name string
    found, err := cache.Get(ctx, "user:1", &name, fallback, nil)
    if err != nil {
        panic(err)
    }

    if found {
        fmt.Println("User name:", name)
    }
}
```

### 使用Redis存储

```go
package main

import (
    "context"
    "fmt"
    "time"

    "go-cache/cacher"
    "go-cache/cacher/store/redis"

    "github.com/redis/go-redis/v9"
)

func main() {
    // 创建Redis客户端
    client := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })

    // 创建Redis存储
    store := redis.NewRedisStore(client)

    // 创建Cacher
    cache := cacher.NewCacher(store)

    ctx := context.Background()

    // 批量获取缓存项
    keys := []string{"user:1", "user:2", "user:3"}
    result := make(map[string]string)
    
    batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
        // 模拟从数据库批量获取数据
        values := make(map[string]interface{})
        for _, key := range keys {
            values[key] = fmt.Sprintf("User %s", key)
        }
        return values, nil
    }

    err := cache.MGet(ctx, keys, &result, batchFallback, nil)
    if err != nil {
        panic(err)
    }

    fmt.Println("Users:", result)
}
```

## 接口设计

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

## 存储后端

### Redis

使用Redis作为存储后端，支持分布式缓存。

### Ristretto

使用Ristretto作为存储后端，支持高性能的内存缓存。

## 测试

项目包含完整的单元测试，确保各个组件的正确性：

```bash
go test ./...
```

## 许可证

MIT