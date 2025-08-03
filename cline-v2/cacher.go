package cacher

import (
	"context"
	"time"
)

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
