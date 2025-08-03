package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go-cache/cacher/store"
)

// RedisStore Redis存储实现
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore 创建新的Redis存储实例
func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

// Get 从Redis获取单个值
func (r *RedisStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// 反序列化
	if err := json.Unmarshal([]byte(val), dst); err != nil {
		return false, err
	}

	return true, nil
}

// MGet 批量获取值到map中
func (r *RedisStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	results, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return err
	}

	// 将结果转换为map[string]interface{}
	resultMap := make(map[string]interface{})
	for i, key := range keys {
		if results[i] != nil {
			var val interface{}
			if err := json.Unmarshal([]byte(results[i].(string)), &val); err != nil {
				return err
			}
			resultMap[key] = val
		}
	}

	// 将结果复制到dstMap
	// 使用类型断言来处理不同的map类型
	switch m := dstMap.(type) {
	case *map[string]interface{}:
		*m = resultMap
	case *map[string]string:
		stringMap := make(map[string]string)
		for k, v := range resultMap {
			if str, ok := v.(string); ok {
				stringMap[k] = str
			} else {
				// 如果值不是字符串，将其转换为字符串
				stringMap[k] = fmt.Sprintf("%v", v)
			}
		}
		*m = stringMap
	default:
		// 对于其他类型，尝试直接赋值
		*(&dstMap) = resultMap
	}

	return nil
}

// Exists 批量检查键存在性
func (r *RedisStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	results, err := r.client.Exists(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	// 检查每个键是否存在
	exists := make(map[string]bool)
	for i, key := range keys {
		exists[key] = (results & (1 << uint(i))) != 0
	}

	return exists, nil
}

// MSet 批量设置键值对，支持TTL
func (r *RedisStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	// 序列化值
	serialized := make(map[string]interface{}, len(items))
	for k, v := range items {
		data, err := json.Marshal(v)
		if err != nil {
			return err
		}
		serialized[k] = string(data)
	}

	// 执行MSet
	if err := r.client.MSet(ctx, serialized).Err(); err != nil {
		return err
	}

	// 设置TTL
	if ttl > 0 {
		pipe := r.client.TxPipeline()
		for key := range items {
			pipe.Expire(ctx, key, ttl)
		}
		cmds, err := pipe.Exec(ctx)
		if err != nil {
			return err
		}
		// 检查每个Expire命令的结果
		for _, cmd := range cmds {
			if cmd.Err() != nil {
				return cmd.Err()
			}
		}
	}

	return nil
}

// Del 删除指定键
func (r *RedisStore) Del(ctx context.Context, keys ...string) (int64, error) {
	result, err := r.client.Del(ctx, keys...).Result()
	if err != nil {
		return 0, err
	}
	return result, nil
}

// 确保RedisStore实现了Store接口
var _ store.Store = (*RedisStore)(nil)