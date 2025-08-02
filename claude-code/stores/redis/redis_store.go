package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	cache "go-cache"
)

// RedisStore Redis存储后端实现
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore 创建Redis存储实例
func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{
		client: client,
	}
}

// Get 从Redis获取单个值
func (r *RedisStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	result, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, fmt.Errorf("redis get failed: %w", err)
	}

	if err := cache.DeserializeValue(result, dst); err != nil {
		return false, fmt.Errorf("deserialize failed: %w", err)
	}

	return true, nil
}

// MGet 批量获取值到map中
func (r *RedisStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	if len(keys) == 0 {
		return nil
	}

	if err := cache.ValidateMapPointer(dstMap); err != nil {
		return err
	}

	results, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return fmt.Errorf("redis mget failed: %w", err)
	}

	for i, result := range results {
		if result != nil {
			if strResult, ok := result.(string); ok {
				if err := cache.SetMapValue(dstMap, keys[i], strResult); err != nil {
					return fmt.Errorf("set map value failed for key %s: %w", keys[i], err)
				}
			}
		}
	}

	return nil
}

// Exists 批量检查键存在性
func (r *RedisStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	if len(keys) == 0 {
		return make(map[string]bool), nil
	}

	pipe := r.client.Pipeline()
	cmds := make([]*redis.IntCmd, len(keys))
	
	for i, key := range keys {
		cmds[i] = pipe.Exists(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("redis exists pipeline failed: %w", err)
	}

	result := make(map[string]bool, len(keys))
	for i, cmd := range cmds {
		exists, err := cmd.Result()
		if err != nil {
			return nil, fmt.Errorf("redis exists failed for key %s: %w", keys[i], err)
		}
		result[keys[i]] = exists > 0
	}

	return result, nil
}

// MSet 批量设置键值对，支持TTL
func (r *RedisStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	pipe := r.client.Pipeline()
	
	for key, value := range items {
		serialized, err := cache.SerializeValue(value)
		if err != nil {
			return fmt.Errorf("serialize failed for key %s: %w", key, err)
		}

		if ttl > 0 {
			pipe.Set(ctx, key, serialized, ttl)
		} else {
			pipe.Set(ctx, key, serialized, 0)
		}
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("redis mset pipeline failed: %w", err)
	}

	return nil
}

// Del 删除指定键
func (r *RedisStore) Del(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	result, err := r.client.Del(ctx, keys...).Result()
	if err != nil {
		return 0, fmt.Errorf("redis del failed: %w", err)
	}

	return result, nil
}