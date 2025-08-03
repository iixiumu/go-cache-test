package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/redis/go-redis/v9"
	"go-cache/cacher/store"
)

// RedisStore 基于Redis的Store实现
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore 创建新的Redis Store
func NewRedisStore(client *redis.Client) store.Store {
	return &RedisStore{client: client}
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

	// 反序列化JSON
	err = json.Unmarshal([]byte(val), dst)
	if err != nil {
		return false, err
	}

	return true, nil
}

// MGet 批量获取值
func (r *RedisStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	if len(keys) == 0 {
		return nil
	}

	// 获取原始值
	vals, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return err
	}

	// 构建结果map
	result := make(map[string]interface{})
	for i, key := range keys {
		if vals[i] != nil {
			var value interface{}
			// 尝试反序列化JSON
			if err := json.Unmarshal([]byte(vals[i].(string)), &value); err != nil {
				// 如果不是JSON，直接使用原始值
				value = vals[i]
			}
			result[key] = value
		}
	}

	// 使用反射设置dstMap
	dstValue := reflect.ValueOf(dstMap)
	if dstValue.Kind() != reflect.Ptr || dstValue.Elem().Kind() != reflect.Map {
		return fmt.Errorf("dstMap must be a pointer to map")
	}

	dstMapValue := dstValue.Elem()
	dstMapValue.Set(reflect.ValueOf(result))

	return nil
}

// Exists 检查键存在性
func (r *RedisStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	if len(keys) == 0 {
		return make(map[string]bool), nil
	}

	// 使用Pipeline批量检查
	pipe := r.client.Pipeline()
	cmds := make([]*redis.IntCmd, len(keys))
	
	for i, key := range keys {
		cmds[i] = pipe.Exists(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[string]bool)
	for i, key := range keys {
		result[key] = cmds[i].Val() > 0
	}

	return result, nil
}

// MSet 批量设置键值对
func (r *RedisStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	// 序列化值
	serialized := make(map[string]interface{})
	for key, value := range items {
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		serialized[key] = string(data)
	}

	// 使用Pipeline批量设置
	pipe := r.client.Pipeline()
	
	for key, value := range serialized {
		if ttl > 0 {
			pipe.Set(ctx, key, value, ttl)
		} else {
			pipe.Set(ctx, key, value, 0)
		}
	}

	_, err := pipe.Exec(ctx)
	return err
}

// Del 删除指定键
func (r *RedisStore) Del(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	return r.client.Del(ctx, keys...).Result()
}
