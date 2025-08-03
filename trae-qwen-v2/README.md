# go-cache

一个Go语言高级缓存库，提供带回退机制的缓存操作。

## 特性

- 支持多种存储后端（Redis、Ristretto等）
- 提供统一的缓存接口
- 支持单个和批量缓存操作
- 支持缓存回退机制
- 支持缓存过期时间（TTL）

## 安装

```bash
go get go-cache
```

## 使用示例

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/dgraph-io/ristretto/v2"
    "go-cache/cacher"
    "go-cache/cacher/store/ristretto"
)

func main() {
    // 创建Ristretto缓存实例
    cache, err := ristretto.NewCache(&ristretto.Config[string, interface{}] {
        NumCounters: 1000,
        MaxCost:     100,
        BufferItems: 64,
    })
    if err != nil {
        log.Fatalf("Failed to create Ristretto cache: %v", err)
    }
    defer cache.Close()

    // 创建RistrettoStore实例
    store := ristretto.NewRistrettoStore(cache)

    // 创建Cacher实例
    cacher := cacher.NewCacherImpl(store)

    // 定义回退函数
    fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
        fmt.Printf("Fetching data for key: %s\n", key)
        // 模拟从数据库或其他数据源获取数据
        data := fmt.Sprintf("Data for %s", key)
        return data, true, nil
    }

    // 使用Get方法获取单个值
    var value string
    found, err := cacher.Get(context.Background(), "key1", &value, fallback, nil)
    if err != nil {
        log.Fatalf("Get failed: %v", err)
    }
    if found {
        fmt.Printf("Got value: %s\n", value)
    }
}
```

## 接口

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

## 测试

```bash
go test ./...
```

## 许可证

MIT