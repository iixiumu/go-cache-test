# Go-Cache

Go-Cache是一个高级Go语言缓存库，提供了业务层缓存抽象。它使用Store作为存储后端，提供更高级的缓存模式和回退策略。

## 特性

- **多存储后端支持**：支持多种存储后端，包括内存、Redis、Ristretto和GCache
- **回退机制**：当缓存未命中时，可以执行回退函数从数据源获取数据
- **批量操作**：支持批量获取、删除和刷新缓存项
- **类型安全**：使用反射实现，支持多种数据类型

## 安装

```bash
go get github.com/xiumu/go-cache
```

## 使用示例

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/xiumu/go-cache/cacher"
	"github.com/xiumu/go-cache/store/memory"
)

func main() {
	// 创建内存存储
	store := memory.NewMemoryStore()

	// 创建缓存器
	cache := cacher.NewCacher(store)

	// 定义回退函数
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		// 模拟从数据库或其他数据源获取数据
		return "value_for_" + key, true, nil
	}

	// 获取缓存项
	var value string
	found, err := cache.Get(context.Background(), "key1", &value, fallback, nil)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	if found {
		fmt.Println("Found value:", value)
	} else {
		fmt.Println("Value not found")
	}
}
```

## 存储后端

### 内存存储

内存存储是默认的存储实现，适用于单机应用或测试环境。

### Redis存储

Redis存储提供了分布式缓存能力，适用于多节点部署的应用。

### Ristretto存储

Ristretto是Dgraph Labs开发的高性能内存缓存，具有高并发和低内存占用的特点。

### GCache存储

GCache是一个支持多种缓存策略（LRU、LFU、ARC）的缓存库。

## 测试

项目包含了完整的单元测试，可以使用以下命令运行测试：

```bash
go test ./...
```