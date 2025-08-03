package redis

import (
	"context"
	"encoding/json"
	"time"

	"go-cache/cacher/store"

	"github.com/redis/go-redis/v9"
)

// RedisStore Redis存储实现
type RedisStore struct {
	client redis.Cmdable
}

// NewRedisStore 创建新的Redis存储实例
func NewRedisStore(client redis.Cmdable) store.Store {
	return &RedisStore{
		client: client,
	}
}

// Get 从Redis获取单个值
func (r *RedisStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}

	// 反序列化JSON到目标变量
	if err := json.Unmarshal([]byte(val), dst); err != nil {
		return false, err
	}

	return true, nil
}

// MGet 批量获取值到map中
func (r *RedisStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	vals, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return err
	}

	// 使用反射来处理dstMap，假设它是指向map[string]T的指针
	result := make(map[string]interface{})
	for i, key := range keys {
		if vals[i] != nil {
			// 检查值是否为string类型
			var stringValue string
			if str, ok := vals[i].(string); ok {
				stringValue = str
			} else {
				// 如果不是string，转换为string
				stringValue = vals[i].(string)
			}

			var value interface{}
			if err := json.Unmarshal([]byte(stringValue), &value); err != nil {
				return err
			}
			result[key] = value
		}
	}

	// 使用反射将result赋值给dstMap
	// 这里需要更复杂的反射逻辑来处理不同的map类型
	// 暂时使用类型断言处理常见的map[string]string情况
	if mapPtr, ok := dstMap.(*map[string]string); ok {
		stringMap := make(map[string]string)
		for k, v := range result {
			if str, ok := v.(string); ok {
				stringMap[k] = str
			}
		}
		*mapPtr = stringMap
		return nil
	}

	// 处理map[string]interface{}情况
	if mapPtr, ok := dstMap.(*map[string]interface{}); ok {
		*mapPtr = result
		return nil
	}

	return nil
}

// Exists 批量检查键存在性
func (r *RedisStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	// Redis EXISTS命令可以一次检查多个键，返回存在的键的数量
	count, err := r.client.Exists(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	// 如果所有键都存在，返回true；否则需要逐个检查
	exists := make(map[string]bool)
	if int(count) == len(keys) {
		// 所有键都存在
		for _, key := range keys {
			exists[key] = true
		}
	} else {
		// 需要逐个检查每个键
		for _, key := range keys {
			keyCount, err := r.client.Exists(ctx, key).Result()
			if err != nil {
				return nil, err
			}
			exists[key] = keyCount > 0
		}
	}

	return exists, nil
}

// MSet 批量设置键值对，支持TTL
func (r *RedisStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	// 将items转换为字符串map以便存储
	stringItems := make(map[string]interface{}, len(items))
	for key, value := range items {
		// 序列化为JSON
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		stringItems[key] = string(data)
	}

	// 执行MSet
	err := r.client.MSet(ctx, stringItems).Err()
	if err != nil {
		return err
	}

	// 如果设置了TTL，为每个键设置过期时间
	if ttl > 0 {
		for key := range items {
			if err := r.client.Expire(ctx, key, ttl).Err(); err != nil {
				return err
			}
		}
	}

	return nil
}

// Del 删除指定键
func (r *RedisStore) Del(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	return r.client.Del(ctx, keys...).Result()
}
