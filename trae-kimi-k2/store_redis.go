package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisStore Redis存储实现
type redisStore struct {
	client *redis.Client
}

// NewRedisStore 创建Redis存储实例
func NewRedisStore(client *redis.Client) Store {
	return &redisStore{
		client: client,
	}
}

// Get 从Redis获取单个值
func (r *redisStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	result, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}
	
	return true, deserializeValue([]byte(result), dst)
}

// MGet 批量获取值
func (r *redisStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	if len(keys) == 0 {
		return nil
	}
	
	results, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return err
	}
	
	// 构建结果map
	data := make(map[string][]byte)
	for i, key := range keys {
		if results[i] != nil {
			if str, ok := results[i].(string); ok {
				data[key] = []byte(str)
			}
		}
	}
	
	return deserializeMap(data, dstMap)
}

// Exists 批量检查键存在性
func (r *redisStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool, len(keys))
	if len(keys) == 0 {
		return result, nil
	}
	
	exists, err := r.client.Exists(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}
	
	for i, key := range keys {
		result[key] = exists > 0 && i < int(exists)
	}
	
	return result, nil
}

// MSet 批量设置键值对
func (r *redisStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}
	
	// 序列化所有值
	serialized, err := serializeMap(items)
	if err != nil {
		return err
	}
	
	// 构建参数
	args := make([]interface{}, 0, len(serialized)*2)
	for key, value := range serialized {
		args = append(args, key, string(value))
	}
	
	// 使用pipeline批量设置
	_, err = r.client.MSet(ctx, args...).Result()
	if err != nil {
		return err
	}
	
	// 设置TTL
	if ttl > 0 {
		pipe := r.client.Pipeline()
		for key := range items {
			pipe.Expire(ctx, key, ttl)
		}
		_, err = pipe.Exec(ctx)
	}
	
	return err
}

// Del 删除指定键
func (r *redisStore) Del(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}
	
	return r.client.Del(ctx, keys...).Result()
}