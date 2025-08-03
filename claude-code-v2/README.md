# Go Cache Library

一个用Go语言实现的高级缓存库，支持多种后端存储和智能回退机制。

## 特性

- **统一接口**: Store接口提供统一的缓存操作API
- **多后端支持**: 支持Redis和Ristretto内存缓存
- **智能回退**: Cacher提供缓存未命中时的回退机制
- **批量操作**: 支持批量获取、设置和删除
- **TTL支持**: 支持过期时间设置
- **类型安全**: 使用反射实现类型安全的缓存操作
- **测试完备**: 提供统一的测试套件，保证各后端一致性

## 架构设计

```
┌─────────────────┐
│     Cacher      │ <- 业务层缓存接口，提供回退机制
├─────────────────┤
│     Store       │ <- 存储接口，统一的缓存操作API
├─────────────────┤
│  Redis Store    │ <- Redis后端实现
│ Ristretto Store │ <- Ristretto内存缓存实现
└─────────────────┘
```

## 快速开始

### 1. Redis Store 示例

```go
package main

import (
    "context"
    "github.com/redis/go-redis/v9"
    "go-cache/cacher/store/redis"
)

func main() {
    ctx := context.Background()
    
    // 创建Redis客户端
    client := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    
    // 创建Redis Store
    store := redis.NewStore(client)
    
    // 设置缓存
    user := User{ID: 1, Name: "Alice", Age: 30}
    err := store.MSet(ctx, map[string]interface{}{"user:1": user}, 0)
    
    // 获取缓存
    var result User
    found, err := store.Get(ctx, "user:1", &result)
    if found {
        fmt.Printf("获取到用户: %+v\\n", result)
    }
}
```

### 2. Ristretto Store 示例

```go
package main

import (
    "context"
    "go-cache/cacher/store/ristretto"
)

func main() {
    ctx := context.Background()
    
    // 创建Ristretto Store
    store, err := ristretto.NewStore()
    if err != nil {
        panic(err)
    }
    defer store.Close()
    
    // 使用方式与Redis Store相同
    user := User{ID: 1, Name: "Alice", Age: 30}
    err = store.MSet(ctx, map[string]interface{}{"user:1": user}, 0)
    
    var result User
    found, err := store.Get(ctx, "user:1", &result)
}
```

### 3. Cacher 高级缓存示例

```go
package main

import (
    "context"
    "go-cache/cacher"
    "go-cache/cacher/store/ristretto"
)

func main() {
    ctx := context.Background()
    
    // 创建Store
    store, _ := ristretto.NewStore()
    defer store.Close()
    
    // 创建Cacher
    cache := cacher.NewCacher(store)
    
    // 定义回退函数
    fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
        // 从数据库或其他数据源获取数据
        user := getUserFromDatabase(key)
        if user != nil {
            return *user, true, nil
        }
        return nil, false, nil
    }
    
    // 获取数据（自动处理缓存未命中）
    var user User
    found, err := cache.Get(ctx, "user:1", &user, fallback, nil)
    if found {
        fmt.Printf("获取到用户: %+v\\n", user)
    }
}
```

## API 文档

### Store 接口

Store接口提供底层存储操作：

```go
type Store interface {
    // 获取单个值
    Get(ctx context.Context, key string, dst interface{}) (bool, error)
    
    // 批量获取值
    MGet(ctx context.Context, keys []string, dstMap interface{}) error
    
    // 检查键存在性
    Exists(ctx context.Context, keys []string) (map[string]bool, error)
    
    // 批量设置键值对
    MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error
    
    // 删除键
    Del(ctx context.Context, keys ...string) (int64, error)
}
```

### Cacher 接口

Cacher接口提供高级缓存功能：

```go
type Cacher interface {
    // 获取缓存项，支持回退函数
    Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error)
    
    // 批量获取缓存项
    MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error
    
    // 批量删除缓存项
    MDelete(ctx context.Context, keys []string) (int64, error)
    
    // 批量刷新缓存项
    MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error
}
```

## 回退机制

### 单个回退函数

```go
type FallbackFunc func(ctx context.Context, key string) (interface{}, bool, error)

fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
    // 从数据库获取数据
    user := database.GetUser(key)
    if user != nil {
        return user, true, nil
    }
    return nil, false, nil
}
```

### 批量回退函数

```go
type BatchFallbackFunc func(ctx context.Context, keys []string) (map[string]interface{}, error)

batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
    users := database.GetUsers(keys)
    result := make(map[string]interface{})
    for k, v := range users {
        result[k] = v
    }
    return result, nil
}
```

## 数据类型支持

库支持多种数据类型的缓存：

- **基础类型**: string, int, bool, float64等
- **结构体**: 自定义结构体（通过JSON序列化，仅Redis）
- **切片和映射**: []string, map[string]int等
- **接口**: interface{}类型

### Redis vs Ristretto

| 特性 | Redis Store | Ristretto Store |
|------|-------------|-----------------|
| 存储方式 | 网络存储 | 内存存储 |
| 序列化 | JSON | 直接对象存储 |
| 持久化 | 支持 | 不支持 |
| 性能 | 网络延迟 | 极快 |
| 内存使用 | 外部 | 进程内 |
| 过期机制 | Redis原生 | 手动实现 |

## 配置选项

### CacheOptions

```go
type CacheOptions struct {
    TTL time.Duration // 缓存过期时间，0表示永不过期
}

opts := &cacher.CacheOptions{
    TTL: 5 * time.Minute,
}
```

### Redis配置

```go
client := redis.NewClient(&redis.Options{
    Addr:     "localhost:6379",
    Password: "", 
    DB:       0,
    PoolSize: 10,
})
```

### Ristretto配置

```go
// 使用默认配置
store, err := ristretto.NewStore()

// 或者自定义配置（需要修改NewStore函数）
cache, err := ristretto.NewCache(&ristretto.Config[string, *cacheItem]{
    NumCounters: 1e7,     // 计数器数量
    MaxCost:     1 << 30, // 最大内存使用
    BufferItems: 64,      // 缓冲区大小
})
```

## 测试

运行所有测试：

```bash
go test ./...
```

运行特定后端测试：

```bash
# Redis Store测试
go test ./cacher/store/redis

# Ristretto Store测试
go test ./cacher/store/ristretto

# Cacher测试
go test ./cacher
```

运行演示：

```bash
go run demo/main.go
```

## 最佳实践

### 1. 选择合适的后端

- **Redis**: 适用于需要持久化、分布式部署的场景
- **Ristretto**: 适用于单机、高性能的场景

### 2. 合理设置TTL

```go
// 短期数据
shortOpts := &cacher.CacheOptions{TTL: 5 * time.Minute}

// 长期数据
longOpts := &cacher.CacheOptions{TTL: 24 * time.Hour}

// 永久缓存
permanentOpts := &cacher.CacheOptions{TTL: 0}
```

### 3. 批量操作优化

```go
// 好的做法：批量操作
keys := []string{"user:1", "user:2", "user:3"}
userMap := make(map[string]User)
err := cache.MGet(ctx, keys, &userMap, batchFallback, nil)

// 避免：循环单个操作
for _, key := range keys {
    var user User
    cache.Get(ctx, key, &user, fallback, nil)
}
```

### 4. 错误处理

```go
found, err := cache.Get(ctx, key, &result, fallback, opts)
if err != nil {
    // 记录错误，但不要阻塞业务逻辑
    log.Printf("缓存错误: %v", err)
    // 可以考虑直接调用fallback
}
```

### 5. 资源管理

```go
// 对于Ristretto，记得关闭
defer store.Close()

// 对于Redis，关闭客户端连接
defer client.Close()
```

## 依赖

- `github.com/redis/go-redis/v9`: Redis客户端
- `github.com/dgraph-io/ristretto/v2`: 内存缓存
- `github.com/stretchr/testify`: 测试框架
- `github.com/alicebob/miniredis/v2`: Redis测试工具

## 许可证

本项目使用MIT许可证。