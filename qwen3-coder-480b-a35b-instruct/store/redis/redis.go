package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/xiumu/go-cache/store"
)

// RedisStore Redis存储实现
type RedisStore struct {
	client redis.Cmdable
}

// New 创建一个新的Redis存储实例
func New(client redis.Cmdable) store.Store {
	return &RedisStore{
		client: client,
	}
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

	// 反序列化JSON到dst
	err = json.Unmarshal([]byte(val), dst)
	if err != nil {
		return false, err
	}

	return true, nil
}

// MGet 批量从Redis获取值
func (r *RedisStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	vals, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return err
	}

	// 将vals转换为dstMap
	// 假设dstMap是*map[string]interface{}类型
	if m, ok := dstMap.(*map[string]interface{}); ok {
		*m = make(map[string]interface{})
		for i, val := range vals {
			if val != nil {
				var data interface{}
				err = json.Unmarshal([]byte(val.(string)), &data)
				if err != nil {
					return err
				}
				(*m)[keys[i]] = data
			}
		}
		return nil
	}

	return nil
}

// Exists 批量检查键存在性
func (r *RedisStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool)
	
	// 逐个检查键是否存在
	for _, key := range keys {
		exists, err := r.client.Exists(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		result[key] = exists > 0
	}

	return result, nil
}

// MSet 批量设置键值对
func (r *RedisStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	// 构造redis的键值对映射
	values := make(map[string]interface{})
	for k, v := range items {
		values[k] = v
	}

	// 执行批量设置
	err := r.client.MSet(ctx, values).Err()
	if err != nil {
		return err
	}

	// 如果设置了TTL，则为每个键设置过期时间
	if ttl > 0 {
		for key := range items {
			err = r.client.Expire(ctx, key, ttl).Err()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Del 删除指定键
func (r *RedisStore) Del(ctx context.Context, keys ...string) (int64, error) {
	return r.client.Del(ctx, keys...).Result()
}