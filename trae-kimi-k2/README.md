# Go Cache

一个功能强大的Go语言高级缓存库，提供业务层缓存抽象和多种存储后端支持。

## 特性

- **高级缓存接口**: 提供带回退机制的缓存操作
- **多种存储后端**: 支持Redis、Ristretto、GCache
- **反射支持**: 支持任意数据类型的序列化和反序列化
- **批量操作**: 支持批量获取、设置、删除操作
- **缓存击穿保护**: 提供带锁的实现防止缓存击穿
- **TTL支持**: 支持缓存过期时间设置
- **回退机制**: 缓存未命中时自动执行回退函数

## 安装

```bash
go get github.com/your-org/go-cache
```

## 快速开始

### 基本使用

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/your-org/go-cache"
    "github.com/redis/go-redis/v9"
)

func main() {
    // 使用Redis存储
    rdb := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    store := cache.NewRedisStore(rdb)
    cacher := cache.NewCacher(store)
    
    ctx := context.Background()
    
    // 获取缓存值（带回退）
    var user User
    found, err := cacher.Get(ctx, "user:123", &user, func(ctx context.Context, key string) (interface{}, bool, error) {
        // 从数据库获取用户
        user, err := getUserFromDB(key)
        if err != nil {
            return nil, false, err
        }
        return user, true, nil
    }, &cache.CacheOptions{TTL: 5 * time.Minute})
    
    if err != nil {
        // 处理错误
    }
    
    if found {
        fmt.Printf("User: %+v\n", user)
    }
}

func getUserFromDB(key string) (*User, error) {
    // 实际的数据库查询逻辑
    return &User{ID: key, Name: "John"}, nil
}

type User struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}
```

### 批量操作

```go
// 批量获取
keys := []string{"user:1", "user:2", "user:3"}
result := make(map[string]User)

err := cacher.MGet(ctx, keys, &result, func(ctx context.Context, keys []string) (map[string]interface{}, error) {
    // 批量从数据库获取
    users, err := getUsersFromDB(keys)
    if err != nil {
        return nil, err
    }
    
    result := make(map[string]interface{})
    for _, user := range users {
        result["user:"+user.ID] = user
    }
    return result, nil
}, &cache.CacheOptions{TTL: 10 * time.Minute})

if err != nil {
    // 处理错误
}
```

### 使用Ristretto内存缓存

```go
import "github.com/dgraph-io/ristretto"

cache, err := ristretto.NewCache(&ristretto.Config{
    NumCounters: 1e7,
    MaxCost:     1 << 30,
    BufferItems: 64,
})
if err != nil {
    panic(err)
}

store := cache.NewRistrettoStore(cache)
cacher := cache.NewCacher(store)
```

### 使用GCache内存缓存

```go
import "github.com/bluele/gcache"

gcache := gcache.New(1000).LRU().Build()
store := cache.NewGCacheStore(gcache)
cacher := cache.NewCacher(store)
```

### 防止缓存击穿

```go
// 使用带锁的Cacher实现
cacher := cache.NewCacherWithLock(store)

// 在高并发场景下，相同的key只会执行一次回退函数
var wg sync.WaitGroup
for i := 0; i < 100; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        var value string
        cacher.Get(ctx, "hot-key", &value, expensiveFallback, nil)
    }()
}
wg.Wait()
```

## 接口说明

### Store接口

底层存储接口，提供基础的键值存储操作：

- `Get(ctx, key, dst)` - 获取单个值
- `MGet(ctx, keys, dstMap)` - 批量获取值
- `Exists(ctx, keys)` - 批量检查键存在性
- `MSet(ctx, items, ttl)` - 批量设置键值对
- `Del(ctx, keys...)` - 删除指定键

### Cacher接口

高级缓存接口，提供带回退机制的缓存操作：

- `Get(ctx, key, dst, fallback, opts)` - 获取单个缓存项
- `MGet(ctx, keys, dstMap, fallback, opts)` - 批量获取缓存项
- `MDelete(ctx, keys)` - 批量清除缓存项
- `MRefresh(ctx, keys, dstMap, fallback, opts)` - 批量强制刷新缓存项

## 支持的存储后端

### Redis
- 基于`go-redis`客户端
- 支持TTL、批量操作
- 支持分布式部署

### Ristretto
- 基于`dgraph-io/ristretto`
- 高性能内存缓存
- 支持LRU/LFU淘汰策略

### GCache
- 基于`bluele/gcache`
- 支持多种淘汰策略（LRU、LFU、ARC）
- 支持最大容量限制

## 测试

```bash
go test -v ./...
```

## 贡献

欢迎提交Issue和Pull Request！

## 许可证

MIT License