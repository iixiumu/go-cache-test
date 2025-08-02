# Go Cache

Go Cache是一个高级缓存库，提供了业务层缓存抽象，内部使用多种存储后端，提供更高级的缓存模式和回退策略。

## 特性

- 抽象的缓存接口设计，易于扩展
- 支持多种存储后端：Redis、Ristretto、GCache
- 支持单个和批量操作
- 支持回退机制，缓存未命中时自动从数据源获取
- 支持TTL过期时间设置
- 使用反射实现，支持多种数据类型

## 安装

```bash
go get github.com/xiumu/go-cache
```

## 使用示例

### 基本使用

```go
package main

import (
    "context"
    "github.com/xiumu/go-cache/cache"
    "github.com/xiumu/go-cache/store/gcache"
    "github.com/bluele/gcache"
)

func main() {
    // 创建一个GCache实例
    gc := gcache.New(1000).LRU().Build()
    
    // 创建GCache存储
    store := gcache.New(gc)
    
    // 创建Cacher实例
    cacher := cache.New(store)
    
    ctx := context.Background()
    
    // 单个获取
    var value string
    found, err := cacher.Get(ctx, "key1", &value, func(ctx context.Context, key string) (interface{}, bool, error) {
        // 缓存未命中时的回退函数
        return "data_from_source", true, nil
    }, nil)
    
    if err != nil {
        // 处理错误
        return
    }
    
    if found {
        // 使用获取到的值
        fmt.Println(value)
    }
}
```

### 批量操作

```go
// 批量获取
keys := []string{"key1", "key2", "key3"}
resultMap := make(map[string]interface{})

err := cacher.MGet(ctx, keys, &resultMap, func(ctx context.Context, keys []string) (map[string]interface{}, error) {
    // 批量回退函数
    result := make(map[string]interface{})
    for _, key := range keys {
        result[key] = getDataFromSource(key) // 从数据源获取数据
    }
    return result, nil
}, nil)

// 批量删除
count, err := cacher.MDelete(ctx, []string{"key1", "key2"})
```

## 存储后端

### Redis

```go
import "github.com/xiumu/go-cache/store/redis"
import "github.com/go-redis/redis/v8"

client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
store := redis.New(client)
```

### Ristretto

```go
import "github.com/xiumu/go-cache/store/ristretto"
import "github.com/dgraph-io/ristretto"

cache, err := ristretto.NewCache(&ristretto.Config{
    NumCounters: 1e7,
    MaxCost:     1 << 30,
    BufferItems: 64,
})
store := ristretto.New(cache)
```

### GCache

```go
import "github.com/xiumu/go-cache/store/gcache"
import "github.com/bluele/gcache"

gc := gcache.New(1000).LRU().Build()
store := gcache.New(gc)
```

## 接口设计

### Store接口

Store是底层存储接口，提供基础的键值存储操作：

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

Cacher是高级缓存接口，提供带回退机制的缓存操作：

```go
type Cacher interface {
    Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error)
    MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error
    MDelete(ctx context.Context, keys []string) (int64, error)
    MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error
}
```

## 许可证

MIT