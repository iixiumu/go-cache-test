package redis

import (
	"context"
	"encoding/json"
	"time"

	"go-cache/cacher/store"

	"github.com/redis/go-redis/v9"
)

// RedisStore 实现了Store接口的Redis存储
type RedisStore struct {
	client redis.Cmdable
}

// NewRedisStore 创建一个新的RedisStore实例
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

	// 反序列化JSON到目标变量
	return true, json.Unmarshal([]byte(val), dst)
}

// MGet 批量从Redis获取值
func (r *RedisStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	result, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return err
	}

	// 创建一个map来存储结果
	resultMap := make(map[string]interface{})

	// 处理结果
	for i, key := range keys {
		if result[i] != nil {
			var value interface{}
			if err := json.Unmarshal([]byte(result[i].(string)), &value); err != nil {
				return err
			}
			resultMap[key] = value
		}
	}

	// 将结果复制到目标map
	dstMapValue := dstMap.(*map[string]interface{})
	*dstMapValue = resultMap

	return nil
}

// Exists 批量检查键在Redis中的存在性
func (r *RedisStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool)

	// Redis的EXISTS命令可以检查多个键
	exists, err := r.client.Exists(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	// 如果所有键都存在
	if int(exists) == len(keys) {
		for _, key := range keys {
			result[key] = true
		}
		return result, nil
	}

	// 逐个检查键的存在性
	for _, key := range keys {
		exists, err := r.client.Exists(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		result[key] = exists == 1
	}

	return result, nil
}

// MSet 批量设置键值对到Redis
func (r *RedisStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	// 将值序列化为JSON
	serializedItems := make(map[string]interface{}, len(items))
	for key, value := range items {
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		serializedItems[key] = string(data)
	}

	// 执行MSet命令
	err := r.client.MSet(ctx, serializedItems).Err()
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

// Del 从Redis删除指定键
func (r *RedisStore) Del(ctx context.Context, keys ...string) (int64, error) {
	return r.client.Del(ctx, keys...).Result()
}
