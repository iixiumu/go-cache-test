# Go Cache Library

一个Go语言高级缓存库，提供带回退机制的缓存操作。

## 项目结构

```
go-cache/
├── cacher/
│   ├── cacher.go          # Cacher接口定义和实现
│   └── cacher_test.go     # Cacher单元测试
├── store/
│   ├── store.go           # Store接口定义
│   ├── store_test.go      # Store接口测试
│   ├── redis/
│   │   ├── redis.go       # Redis Store实现
│   │   └── redis_test.go  # Redis Store测试
│   ├── ristretto/
│   │   ├── ristretto.go   # Ristretto Store实现
│   │   └── ristretto_test.go
│   └── gcache/
│       ├── gcache.go      # GCache Store实现
│       └── gcache_test.go
├── go.mod
└── README.md
```

## 核心组件

### Store 接口

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

### Cacher 接口

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

## 功能特性

1. **多存储后端支持**：
   - Redis
   - Ristretto
   - GCache

2. **回退机制**：
   - 单个键回退函数
   - 批量键回退函数

3. **批量操作**：
   - 批量获取
   - 批量删除
   - 批量刷新

4. **TTL支持**：
   - 可为缓存项设置过期时间

## 使用示例

```go
// 创建存储后端实例
redisStore := redis.NewRedisStore(redisClient)

// 创建缓存器
cacher := cacher.NewCacher(redisStore)

// 定义回退函数
fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
    // 从数据库或其他数据源获取数据
    value, err := getDataFromDatabase(key)
    if err != nil {
        return nil, false, err
    }
    return value, true, nil
}

// 获取缓存项
var result string
found, err := cacher.Get(context.Background(), "key", &result, fallback, nil)
if err != nil {
    // 处理错误
}

// 批量获取缓存项
keys := []string{"key1", "key2", "key3"}
resultMap := make(map[string]string)
err = cacher.MGet(context.Background(), keys, &resultMap, batchFallback, nil)
if err != nil {
    // 处理错误
}
```

## 测试

项目为各个组件提供了单元测试：

- `cacher/cacher_test.go`：Cacher接口测试
- `store/store_test.go`：Store接口测试
- `store/redis/redis_test.go`：Redis Store测试
- `store/ristretto/ristretto_test.go`：Ristretto Store测试
- `store/gcache/gcache_test.go`：GCache Store测试

## 依赖

项目使用了以下第三方库：

- `github.com/go-redis/redis/v8`：Redis客户端
- `github.com/hypermodeinc/ristretto`：Ristretto缓存库
- `github.com/bluele/gcache`：GCache缓存库
- `github.com/alicebob/miniredis/v2`：Redis测试工具

## 实现细节

1. **反射处理**：由于缓存库要支持多种数据类型，使用反射实现而不是泛型。

2. **接口设计**：通过Store接口抽象底层存储，使得可以轻松切换不同的存储后端。

3. **回退机制**：当缓存未命中时，可以执行回退函数从数据源获取数据并自动缓存。

4. **批量操作**：支持批量获取、删除和刷新操作，提高性能。
