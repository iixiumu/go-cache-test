# Go高级缓存库

这是一个功能丰富的Go语言高级缓存库，提供了灵活的缓存抽象和多种存储后端实现。

## 项目结构

```
/pkg
  /cacher      # 高级缓存接口和实现
  /store       # 底层存储接口
    /redis     # Redis存储实现
    /ristretto # Ristretto存储实现
    /gcache    # GCache存储实现
/test         # 单元测试
/main.go      # 示例程序
```

## 核心接口

### Store接口
底层存储接口，提供基础的键值存储操作：
- `Get`: 获取单个值
- `MGet`: 批量获取值
- `Exists`: 批量检查键存在性
- `MSet`: 批量设置键值对，支持TTL
- `Del`: 删除指定键

### Cacher接口
高级缓存接口，提供带回退机制的缓存操作：
- `Get`: 获取单个缓存项，缓存未命中时执行回退函数
- `MGet`: 批量获取缓存项，支持部分命中和批量回退
- `MDelete`: 批量清除缓存项
- `MRefresh`: 批量强制刷新缓存项

## 存储后端实现

1. **Redis**: 基于Redis的分布式缓存实现
2. **Ristretto**: 基于DGraph Ristretto的高性能内存缓存实现
3. **GCache**: 基于Bluele GCache的内存缓存实现

## 安装

```bash
go get github.com/xiumu/git/me/go-cache/trae
```

## 使用示例

### 初始化Redis存储后端

```go
import (
	"github.com/go-redis/redis/v8"
	"github.com/xiumu/git/me/go-cache/trae/pkg/store/redis"
	"github.com/xiumu/git/me/go-cache/trae/pkg/cacher"
)

// 创建Redis客户端
client := redis.NewClient(&redis.Options{
	Addr: "localhost:6379",
})

// 创建RedisStore
redisStore := redis.NewRedisStore(client)

// 创建Cacher
c := cacher.NewCacher(redisStore)
```

### 使用缓存

```go
import "context"

ctx := context.Background()

// 获取单个缓存项
var val string
found, err := c.Get(ctx, "key", &val, func(ctx context.Context, key string) (interface{}, bool, error) {
	// 回退函数，从数据源获取数据
	return "value", true, nil
}, nil)

// 批量获取缓存项
results := make(map[string]interface{})
keys := []string{"key1", "key2", "key3"}
err = c.MGet(ctx, keys, &results, func(ctx context.Context, keys []string) (map[string]interface{}, error) {
	// 批量回退函数
	result := make(map[string]interface{})
	for _, key := range keys {
		result[key] = "value-" + key
	}
	return result, nil
}, nil)

// 设置缓存过期时间
opts := &cacher.CacheOptions{
	TTL: 5 * time.Minute,
}
var valWithTTL string
found, err = c.Get(ctx, "key-with-ttl", &valWithTTL, fallback, opts)
```

## 运行测试

```bash
# 运行所有测试
go test ./test/...

# 运行特定测试
go test ./test/ -run TestRedisStore
```

## 运行示例程序

```bash
# 编译并运行示例程序
go run main.go
```