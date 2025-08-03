package store

import (
	"context"
	"time"
)

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
