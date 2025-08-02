# Go-Cache - 高级Go缓存库

Go-Cache是一个功能强大的Go语言缓存库，提供了高级缓存模式和回退策略。它支持多种存储后端，包括Redis、Ristretto和GCache。

## 特性

- **多种存储后端支持**：Redis、Ristretto、GCache
- **回退机制**：缓存未命中时自动从数据源获取数据
- **批量操作**：支持批量获取、设置和删除
- **TTL支持**：可为缓存项设置过期时间
- **泛型支持**：通过反射支持多种数据类型
- **接口设计**：易于扩展和测试

## 安装

```bash
go get github.com/xiumu/go-cache
```

## 快速开始

### 1. 创建存储实例

```go
// 使用Ristretto作为存储后端
cache, err := ristretto.NewCache(&ristretto.Config{
    NumCounters: 1000,
    MaxCost:     10000,
    BufferItems: 64,
})
if err != nil {
    panic(err)
}

store := store.NewRistrettoStore(cache)
```

### 2. 创建缓存实例

```go
cacher := cache.New(store)
```

### 3. 使用缓存

```go
ctx := context.Background()

// 单个获取
var user User
found, err := cacher.Get(ctx, "user:1", &user, func(ctx context.Context, key string) (interface{}, bool, error) {
    // 回退函数，当缓存未命中时调用
    user, err := getUserFromDatabase(key)
    if err != nil {
        return nil, false, err
    }
    return user, true, nil
}, nil)

// 批量获取
keys := []string{"user:1", "user:2", "user:3"}
users := make(map[string]User)
err := cacher.MGet(ctx, keys, &users, func(ctx context.Context, keys []string) (map[string]interface{}, error) {
    // 批量回退函数
    return getUsersFromDatabase(keys)
}, nil)
```

## 存储后端

### Redis

```go
import "github.com/go-redis/redis/v8"

client := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})

store := store.NewRedisStore(client)
```

### Ristretto

```go
import "github.com/dgraph-io/ristretto"

cache, err := ristretto.NewCache(&ristretto.Config{
    NumCounters: 1000,
    MaxCost:     10000,
    BufferItems: 64,
})

store := store.NewRistrettoStore(cache)
```

### GCache

```go
import "github.com/bluele/gcache"

gcache := gcache.New(1000).LRU().Build()
store := store.NewGCacheStore(gcache)
```

## API参考

### Cacher接口

```go
type Cacher interface {
    // Get 获取单个缓存项
    Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error)
    
    // MGet 批量获取缓存项
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
    
    // MSet 批量设置键值对
    MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error
    
    // Del 删除指定键
    Del(ctx context.Context, keys ...string) (int64, error)
}
```

## 测试

运行所有测试：

```bash
go test ./...
```

## 许可证

MIT