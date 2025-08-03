package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/redis/go-redis/v9"

)

// RedisStore Redis存储实现
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
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("redis get failed: %w", err)
	}

	// 反序列化JSON到目标变量
	if err := json.Unmarshal([]byte(result), dst); err != nil {
		return false, fmt.Errorf("json unmarshal failed: %w", err)
	}

	return true, nil
}

// MGet 批量获取值到map中
func (r *RedisStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	if len(keys) == 0 {
		return nil
	}

	results, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return fmt.Errorf("redis mget failed: %w", err)
	}

	// 创建结果map
	resultMap := make(map[string]interface{})
	for i, key := range keys {
		if results[i] != nil {
			// 反序列化JSON
			var value interface{}
			if err := json.Unmarshal([]byte(results[i].(string)), &value); err != nil {
				continue // 跳过反序列化失败的值
			}
			resultMap[key] = value
		}
	}

	// 使用反射设置目标map
	ptr := reflect.ValueOf(dstMap)
	if ptr.Kind() != reflect.Ptr {
		return fmt.Errorf("dstMap must be a pointer to map[string]interface{}")
	}
	
	mapVal := ptr.Elem()
	if mapVal.Kind() != reflect.Map {
		return fmt.Errorf("dstMap must be a pointer to map[string]interface{}")
	}

	for k, v := range resultMap {
		mapVal.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v))
	}

	return nil
}

// Exists 批量检查键存在性
func (r *RedisStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	if len(keys) == 0 {
		return make(map[string]bool), nil
	}

	results := make(map[string]bool)
	
	// 使用pipeline提高效率
	pipe := r.client.Pipeline()
	cmds := make([]*redis.IntCmd, len(keys))
	
	for i, key := range keys {
		cmds[i] = pipe.Exists(ctx, key)
	}
	
	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("redis pipeline exec failed: %w", err)
	}

	for i, key := range keys {
		if cmds[i].Err() == nil {
			results[key] = cmds[i].Val() > 0
		} else {
			results[key] = false
		}
	}

	return results, nil
}

// MSet 批量设置键值对，支持TTL
func (r *RedisStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	// 使用pipeline提高效率
	pipe := r.client.Pipeline()
	
	for key, value := range items {
		// 序列化值为JSON
		jsonData, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("json marshal failed for key %s: %w", key, err)
		}
		
		if ttl > 0 {
			pipe.Set(ctx, key, string(jsonData), ttl)
		} else {
			pipe.Set(ctx, key, string(jsonData), 0)
		}
	}
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("redis pipeline exec failed: %w", err)
	}

	return nil
}

// Del 删除指定键
func (r *RedisStore) Del(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	deleted, err := r.client.Del(ctx, keys...).Result()
	if err != nil {
		return 0, fmt.Errorf("redis del failed: %w", err)
	}

	return deleted, nil
}

// Close 关闭Redis连接
func (r *RedisStore) Close() error {
	return r.client.Close()
}