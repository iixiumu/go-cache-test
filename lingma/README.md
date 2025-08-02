# Go-Cache - 高级缓存库

Go-Cache是一个Go语言实现的高级缓存库，提供了业务层缓存抽象。它通过[Cacher](file:///Users/xiumu/git/me/go-cache/lingma/cache.go#L47-L66)接口提供高级缓存模式和回退策略，并使用[Store](file:///Users/xiumu/git/me/go-cache/lingma/cache.go#L8-L28)接口作为底层存储后端。

## 特性

- **抽象设计**：通过[Cacher](file:///Users/xiumu/git/me/go-cache/lingma/cache.go#L47-L66)和[Store](file:///Users/xiumu/git/me/go-cache/lingma/cache.go#L8-L28)接口实现关注点分离
- **回退机制**：支持单个和批量回退函数，在缓存未命中时自动加载数据
- **多种存储后端**：可基于Redis、Ristretto、GCache等实现Store接口
- **灵活的缓存策略**：支持TTL过期时间和批量操作
- **反射实现**：通过反射支持多种数据类型，避免使用复杂的泛型

## 核心接口

### Store接口

[Store](file:///Users/xiumu/git/me/go-cache/lingma/cache.go#L8-L28)是底层存储接口，提供基础的键值存储操作：

- `Get`: 获取单个值
- `MGet`: 批量获取值
- `Exists`: 批量检查键存在性
- `MSet`: 批量设置键值对
- `Del`: 删除指定键

### Cacher接口

[Cacher](file:///Users/xiumu/git/me/go-cache/lingma/cache.go#L47-L66)是高级缓存接口，提供带回退机制的缓存操作：

- `Get`: 获取单个缓存项，支持回退函数
- `MGet`: 批量获取缓存项，支持部分命中和批量回退
- `MDelete`: 批量清除缓存项
- `MRefresh`: 批量强制刷新缓存项

## 使用示例

```go
// 创建存储后端（以内存存储为例）
store := cache.NewMemoryStore()

// 创建缓存器
cacher := cache.NewCacher(store)

// 定义回退函数
fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
    // 从数据源获取数据
    value, err := fetchFromDataSource(key)
    if err != nil {
        return nil, false, err
    }
    return value, true, nil
}

// 获取数据
var result string
found, err := cacher.Get(context.Background(), "key", &result, fallback, nil)
```

## 测试

运行测试：

```bash
go test -v ./...
```