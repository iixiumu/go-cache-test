# Go Cache 高级缓存库

这是一个Go语言的高级缓存库，提供了业务层缓存抽象，内部使用Store作为存储后端，支持多种缓存模式和回退策略。

## 特性

- **高级缓存抽象**: Cacher接口提供带回退机制的缓存操作
- **多种存储后端**: 支持Redis和Ristretto两种存储后端
- **批量操作**: 支持批量获取、设置、删除和刷新
- **TTL支持**: Redis后端支持TTL过期时间
- **类型安全**: 使用反射实现类型安全的缓存操作
- **统一测试**: 为所有Store实现提供统一的测试套件

## 架构设计

```
Cacher (业务层缓存抽象)
    ↓
Store (存储后端接口)
    ↓
Redis Store / Ristretto Store (具体实现)
```

### Cacher接口

提供高级缓存操作，包括：

- `Get`: 获取单个缓存项，支持回退函数
- `MGet`: 批量获取缓存项，支持部分命中和批量回退
- `MDelete`: 批量清除缓存项
- `MRefresh`: 批量强制刷新缓存项

### Store接口

提供底层存储操作，包括：

- `Get`: 获取单个值
- `MGet`: 批量获取值
- `Exists`: 检查键存在性
- `MSet`: 批量设置键值对
- `Del`: 删除指定键

## 安装

```bash
go get github.com/your-repo/go-cache
```

## 使用方法

### 使用Redis后端

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/redis/go-redis/v9"
    "go-cache/cacher"
    redisStore "go-cache/cacher/store/redis"
)

func main() {
    // 创建Redis客户端
    client := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    defer client.Close()

    // 创建Redis Store和Cacher
    store := redisStore.NewRedisStore(client)
    cacher := cacher.NewCacher(store)

    // 创建回退函数
    fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
        // 从数据库获取数据
        if key == "user:1" {
            return "Alice", true, nil
        }
        return nil, false, nil
    }

    ctx := context.Background()

    // 获取缓存项
    var user string
    found, err := cacher.Get(ctx, "user:1", &user, fallback, nil)
    if err != nil {
        log.Fatal(err)
    }
    if found {
        fmt.Printf("用户: %s\n", user)
    }
}
```

### 使用Ristretto后端

```go
package main

import (
    "context"
    "fmt"
    "log"

    "go-cache/cacher"
    ristrettoStore "go-cache/cacher/store/ristretto"
)

func main() {
    // 创建Ristretto Store和Cacher
    store, err := ristrettoStore.NewRistrettoStore()
    if err != nil {
        log.Fatal(err)
    }

    cacher := cacher.NewCacher(store)

    // 创建回退函数
    fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
        // 从数据库获取数据
        if key == "product:1" {
            return map[string]interface{}{
                "id":    1,
                "name":  "iPhone 15",
                "price": 999.99,
            }, true, nil
        }
        return nil, false, nil
    }

    ctx := context.Background()

    // 获取缓存项
    var product map[string]interface{}
    found, err := cacher.Get(ctx, "product:1", &product, fallback, nil)
    if err != nil {
        log.Fatal(err)
    }
    if found {
        fmt.Printf("产品: %+v\n", product)
    }
}
```

### 批量操作

```go
// 批量获取
var resultMap map[string]interface{}
batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
    result := make(map[string]interface{})
    for _, key := range keys {
        // 从数据库批量获取数据
        if key == "user:1" {
            result[key] = "Alice"
        }
    }
    return result, nil
}

err := cacher.MGet(ctx, []string{"user:1", "user:2"}, &resultMap, batchFallback, nil)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("结果: %v\n", resultMap)

// 批量删除
deleted, err := cacher.MDelete(ctx, []string{"user:1", "user:2"})
if err != nil {
    log.Fatal(err)
}
fmt.Printf("删除了 %d 个键\n", deleted)

// 批量刷新
err = cacher.MRefresh(ctx, []string{"user:1"}, &resultMap, batchFallback, nil)
if err != nil {
    log.Fatal(err)
}
```

### 使用TTL

```go
// 设置TTL
opts := &cacher.CacheOptions{TTL: 5 * time.Minute}
found, err := cacher.Get(ctx, "user:1", &user, fallback, opts)
```

## 测试

运行所有测试：

```bash
go test ./...
```

运行特定包的测试：

```bash
go test ./cacher/store/redis/...
go test ./cacher/store/ristretto/...
go test ./cacher/...
```

## 实现细节

### Redis Store

- 使用 `github.com/redis/go-redis/v9` 客户端
- 支持JSON序列化和反序列化
- 支持TTL过期时间
- 使用Pipeline进行批量操作

### Ristretto Store

- 使用 `github.com/dgraph-io/ristretto/v2` 缓存
- 直接存储对象，不进行序列化
- 支持TTL过期时间
- 原生线程安全（无需额外锁）
- 高性能内存缓存

### 统一测试套件

为所有Store实现提供统一的测试套件，包括：

- 基本Get/Set操作
- 批量操作
- 键存在性检查
- 删除操作
- TTL功能（Redis）
- 复杂类型支持
- 键不存在的情况

## 注意事项

1. **Redis后端**: 使用JSON序列化，支持TTL
2. **Ristretto后端**: 直接存储对象，支持TTL，原生线程安全
3. **类型安全**: 使用反射实现类型转换
4. **线程安全**: 所有实现都是线程安全的
5. **错误处理**: 缓存失败不影响业务逻辑

## 贡献

欢迎提交Issue和Pull Request！

## 许可证

MIT License 