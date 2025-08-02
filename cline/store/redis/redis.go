package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/example/go-cache/store"
	"github.com/go-redis/redis/v8"
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
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// 反序列化到dst
	err = json.Unmarshal([]byte(val), dst)
	if err != nil {
		return false, err
	}

	return true, nil
}

// MGet 批量从Redis获取值
func (r *RedisStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	result, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return err
	}

	// 将结果转换为map[string]interface{}
	m := make(map[string]interface{})
	for i, key := range keys {
		if result[i] != nil {
			var val interface{}
			err := json.Unmarshal([]byte(result[i].(string)), &val)
			if err != nil {
				return err
			}
			m[key] = val
		}
	}

	// 使用反射将结果赋值给dstMap
	// 这里简化处理，实际实现可能需要更复杂的反射逻辑
	// 假设dstMap是指向map[string]interface{}的指针
	dst := dstMap.(*map[string]interface{})
	*dst = m

	return nil
}

// Exists 批量检查键存在性
func (r *RedisStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result, err := r.client.Exists(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	// 如果所有键都存在或都不存在
	exists := make(map[string]bool)
	if result == int64(len(keys)) {
		// 所有键都存在
		for _, key := range keys {
			exists[key] = true
		}
	} else if result == 0 {
		// 所有键都不存在
		for _, key := range keys {
			exists[key] = false
		}
	} else {
		// 部分键存在，需要逐个检查
		for _, key := range keys {
			val, err := r.client.Exists(ctx, key).Result()
			if err != nil {
				return nil, err
			}
			exists[key] = val > 0
		}
	}

	return exists, nil
}

// MSet 批量设置键值对
func (r *RedisStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	// 将items转换为map[string]interface{}用于MSet
	values := make(map[string]interface{}, len(items))
	for k, v := range items {
		// 序列化值
		data, err := json.Marshal(v)
		if err != nil {
			return err
		}
		values[k] = string(data)
	}

	// 执行MSet
	err := r.client.MSet(ctx, values).Err()
	if err != nil {
		return err
	}

	// 如果设置了TTL，为每个键设置过期时间
	if ttl > 0 {
		for key := range items {
			err := r.client.Expire(ctx, key, ttl).Err()
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
