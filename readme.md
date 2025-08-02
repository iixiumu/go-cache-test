# readme

随便测试了一下市面上主流的AI辅助编程工具，配合不同的模型

## 测试步骤

- 场景：实现go语言高级缓存库，Cacher是业务层缓存抽象，内部使用Store作为存储后端，提供更高级的缓存模式和回退策略
- claude desktop确定接口
- 各种测试

## prompt

实现go语言高级缓存库，Cacher是业务层缓存抽象，内部使用Store作为存储后端，提供更高级的缓存模式和回退策略

```go
// Store 底层存储接口，提供基础的键值存储操作
type Store interface {
    // Get 从存储后端获取单个值
    // key: 键名
    // dst: 目标变量的指针，用于接收反序列化后的值
    // 返回: 是否找到该键, 错误信息
    Get(ctx context.Context, key string, dst interface{}) (bool, error)

    // MGet 批量获取值到map中
    // keys: 要获取的键列表
    // dstMap: 目标map的指针，用于接收结果，类型为*map[string]T
    // 返回: 错误信息
    MGet(ctx context.Context, keys []string, dstMap interface{}) error

    // Exists 批量检查键存在性
    // keys: 要检查的键列表
    // 返回: map[string]bool 键存在性映射, 错误信息
    Exists(ctx context.Context, keys []string) (map[string]bool, error)

    // MSet 批量设置键值对，支持TTL
    // items: 键值对映射
    // ttl: 过期时间，0表示永不过期
    // 返回: 错误信息
    MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error

    // Del 删除指定键
    // keys: 要删除的键列表
    // 返回: 实际删除的键数量, 错误信息
    Del(ctx context.Context, keys ...string) (int64, error)
}

// FallbackFunc 回退函数类型
// 当缓存未命中时执行，用于从数据源获取数据
// key: 请求的键
// 返回: 获取到的值, 是否找到, 错误信息
type FallbackFunc func(ctx context.Context, key string) (interface{}, bool, error)

// BatchFallbackFunc 批量回退函数类型
// 当批量缓存部分未命中时执行，用于从数据源批量获取数据
// keys: 未命中的键列表
// 返回: 键值映射, 错误信息
type BatchFallbackFunc func(ctx context.Context, keys []string) (map[string]interface{}, error)

// CacheOptions 缓存选项
type CacheOptions struct {
    // TTL 缓存过期时间，0表示永不过期
    TTL time.Duration
}

// Cacher 高级缓存接口，提供带回退机制的缓存操作
type Cacher interface {
    // Get 获取单个缓存项，缓存未命中时执行回退函数并缓存结果
    // key: 键名
    // dst: 目标变量的指针，用于接收值
    // fallback: 缓存未命中时的回退函数
    // opts: 缓存选项，可以为nil使用默认选项
    // 返回: 是否找到值（包括从回退函数获取）, 错误信息
    Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error)

    // MGet 批量获取缓存项，支持部分命中和批量回退
    // keys: 要获取的键列表
    // dstMap: 目标map的指针，用于接收结果，类型为*map[string]T
    // fallback: 批量回退函数，处理未命中的键
    // opts: 缓存选项，可以为nil使用默认选项
    // 返回: 错误信息
    MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error

    // MDelete 批量清除缓存项
    // keys: 要删除的键列表
    // 返回: 实际删除的键数量, 错误信息
    MDelete(ctx context.Context, keys []string) (int64, error)

    // MRefresh 批量强制刷新缓存项
    // keys: 要刷新的键列表
    // dstMap: 目标map的指针，用于接收结果，类型为*map[string]T
    // fallback: 批量回退函数
    // opts: 缓存选项，可以为nil使用默认选项
    // 返回: 错误信息
    MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error
}
```

- 为Cacher设计单元测试
- 实现Cacher接口
- 为Store设计单元测试
- 分别基于redis、hypermodeinc/ristretto、bluele/gcache实现Store接口
- redis测试可以用alicebob/miniredis
- 由于缓存库要支持多种数据类型，用泛型使用很麻烦，所以用反射实现

## 测试结果

先说一下普遍存在的问题
    
    go的版本，普遍比较老
    redis依赖，普遍还在用github.com/redis/go-redis/v8 v9
    内存缓存库，大多还做了序列化反序列化来存取

不过这些问题可以通过提示词，MCP优化一下

### claude desktop

通过claude desktop确定了接口定义，顺便实现了Cacher接口，测试没通过

简评：难评，可能是因为不能执行代码调试吧

### Claude Code

- claude

丝滑

简评：claude code yyds

- qwen3-coder-plus

通过claude-code-router接入，效果还不错

简评：阿里牛逼

- qwen3-coder-flash

写出来了，单测不通过

简评：可能是难为他了

- qwen3-coder-480b-a35b-instruct

写出来了，单测不通过

简评：略失望

- qwen3-235b-a22b-instruct-2507

没写完

简评：可能是难为他了

- qwen3-235b-a22b-thinking-2507

写出来了，单测不通过

简评：可能是难为他了

### Gemini CLI

丝滑

简评：很厉害了

### Augment

丝滑

简评：白嫖50次，完成任务只用了一次，牛逼

### Qwen Code

- qwen3-coder-480b-a35b-instruct

未完成

简评：从Gemini CLI改的，感觉更多是模型的原因

### cursor

auto模式，第一版测试没有通过，修了一次通过了

简评：cursor auto模式用的模型太低级了，能完成也还可以了

### Trae-CN

- doubao1.6

未完成

简评：吹过了

- kimi k2

未完成

简评：吹过了

- deepseek v3 & r1

未完成

简评：可能力不从心了

### 通义灵码

完成任务

简评：白嫖Qwen3-Coder

### kiro

完成任务

简评：白嫖claude

### cline

- qwen3-coder-480b-a35b-instruct

未完成

### roocode

- qwen3-coder-480b-a35b-instruct

完成任务

### kilocode

- qwen3-coder-480b-a35b-instruct

完成任务

## gemini给的评价

    结论与建议

    综合来看，`cursor` 和 `Gemini CLI` 的实现质量最高，可以作为非常优秀的基础代码。

    * `cursor` 的优势在于其完整性、出色的文档（注释）和对多种后端存储的实现。它的代码几乎可以直接用于生产环境。
    * `Gemini CLI` 的优势在于其优秀的代码结构（使用了 `internal` 包）和高质量的测试（包含基准测试），这更符合Go社区的最佳实践，长期维护性可能更好。

    `claude-code` 是一个可靠的备选项，它的错误处理和测试模拟方式值得学习。

    `qwen3-235b-a22b-thinking-2507` 的实现则相对粗糙，特别是缺乏测试，不推荐使用。

    最终建议:

    如果需要一个功能最全、开箱即用的库，选择 `cursor`。
    如果想在一个结构最规范、长期维护性最好的基础上进行二次开发，选择 `Gemini CLI`。

## 主观评价

### 工具角度

- Claude Code

YYDS

一个月20多刀也还行，但是封号很严重，使用成本太高了，而且最近调整了使用次数限制

配合claude-code-router用Qwen3-Coder还不错

- Gemini CLI

大善人，稍长上下文+超多请求次数

- Augment

很强，很贵，可以白嫖50次，付费50刀可以用600次

不过话又说回来了，贵不是他的缺点，是我的

- cursor

综合下来体验还可以，各方面比较均衡，cursor前段时间改了收费模式，无限次auto，一些人只看到了无限次被扣费。定价模式很坑，不过我依然感觉是当下比较均衡的一个选择。前端时间对国内禁用claude模型了，我才知道原来这玩意儿之前不挂代理也可以用的，不过感觉这个不是什么大问题，干这行的谁还没个🪜呢。

- Qwen Code

从Gemini CLI改的，刚看到已经更新到v0.0.4了(但是今天周六啊)

最早上线的时候，很多人白嫖百炼的一百万次qwen3-coder-plus请求，然后大面积被反薅羊毛。猜测几个原因qwen3-coder-plus超长上下文太贵了，缓存做的很差，Qwen Code可能也有些问题

- 通义灵码

白嫖Qwen3-Coder，但是不知道嫖的是哪款，效果还不错，cursor平替

- cline，roocode，kilocode

roocode基于cline，kilocode基于cline+roocode

感觉cline差点儿意思，kilocode用力过猛，toocode正好合适

- kiro

很先进，但是不适合国情，白嫖claude

- Trae-CN

国内版配合各模型效果都很差，而且默认是ask模式，找了半天才改成agent模式，体验很糟糕

之前国外版能白嫖claude，不过现在收费了，不过比cursor便宜

solo模式据说很厉害，不过没排到

- CodeBuddy

鹅厂出品

插件上次更新还在2025-05-30，很不积极，可以白嫖deepseek

IDE需要邀请码

- Comate

熊厂出品

有插件，也有IDE，没用过，不评价

### 模型角度

- claude

跑分没赢过，体验没输过

- gemini

2.5 pro性能可以，搭配gemnni-cli食用更佳

- qwen3-coder系列

阿里最近连续发布了一些列大模型，成了真正的OpenAI了（原本的OpenAI已经改名CloseAI了），再联系Facebook刚宣布以后不开源了，阿里真是大善人，最好的开源在东方了！

qwen3-coder-flash效果还是查一些，可能还是规模太小了。qwen3-coder-plus确实不错，个人感觉比qwen3-coder-480b-a35b-instruct强一些，但是同样上下文下，plsu比480便宜，想不明白。

qwen3-coder-plus刚上线那会儿，薅了好多来薅羊毛的羊的毛，然后紧急宣布折扣，1M上下文确实贵。

阿里有个魔搭社区，每天提供2000次免费调用，大善人，这个倒是可以薅。

- glm4.5

感觉一般

- kimi k2

感觉一般

## 总结

本次测试制造的所有垃圾 [go-cache-test](https://github.com/iixiumu/go-cache-test)

```text
───────────────────────────────────────────────────────────────────────────────
Language                 Files     Lines   Blanks  Comments     Code Complexity
───────────────────────────────────────────────────────────────────────────────
Go                         206     28389     4781      3827    19781       4463
Markdown                    48      5114     1106         0     4008          0
JSON                         8       109        0         0      109          0
Makefile                     1        52       10        10       32          1
───────────────────────────────────────────────────────────────────────────────
Total                      263     33664     5897      3837    23930       4464
───────────────────────────────────────────────────────────────────────────────
Estimated Cost to Develop (organic) $757,678
Estimated Schedule Effort (organic) 12.38 months
Estimated People Required (organic) 5.44
───────────────────────────────────────────────────────────────────────────────
Processed 865894 bytes, 0.866 megabytes (SI)
───────────────────────────────────────────────────────────────────────────────
```

个人感觉当前最重要的还是模型能力，至于Agent，我比较认同[How to Build an Agent](https://ampcode.com/how-to-build-an-agent)中的一句话————It’s an LLM, a loop, and enough tokens. 

最后叠个甲，本次测试主要测试无人工干预，一次完成的效果，相信优化提示词，再加上一些MCP，效果都会比现在好很多

## 链接

go-cache-test https://github.com/iixiumu/go-cache-test

How to Build an Agent https://ampcode.com/how-to-build-an-agent