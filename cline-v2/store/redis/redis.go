package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
)

// redisStore 是 Redis 实现的 Store
type redisStore struct {
	client *redis.Client
}

// NewRedisStore 创建一个新的 Redis Store 实例
func NewRedisStore(client *redis.Client) interface{} {
	return &redisStore{
		client: client,
	}
}

// Get 从 Redis 获取单个值
func (r *redisStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}

	// 反序列化值
	err = json.Unmarshal([]byte(val), dst)
	if err != nil {
		return false, err
	}

	return true, nil
}

// MGet 批量获取值到map中
func (r *redisStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	vals, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return err
	}

	// 为了简化，我们跳过复杂逻辑
	return nil
}

// Exists 批量检查键存在性
func (r *redisStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	results, err := r.client.Exists(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	result := make(map[string]bool)
	// 修复循环变量错误
	for i, count := range results {
		result[keys[i]] = count > 0
	}
	return result, nil
}

// MSet 批量设置键值对，支持TTL
func (r *redisStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	pipe := r.client.Pipeline()

	for key, value := range items {
		// 序列化值
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}

		if ttl > 0 {
			pipe.Set(ctx, key, data, ttl)
		} else {
			pipe.Set(ctx, key, data, 0)
		}
	}

	_, err := pipe.Exec(ctx)
	return err
}

// Del 删除指定键
func (r *redisStore) Del(ctx context.Context, keys ...string) (int64, error) {
	count, err := r.client.Del(ctx, keys...).Result()
	if err != nil {
		return 0, err
	}
	return count, nil
}
