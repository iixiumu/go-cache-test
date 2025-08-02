package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/xiumu/go-cache/cache"
)

// RedisStore 实现Store接口，基于Redis
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore 创建新的RedisStore实例
func NewRedisStore(client *redis.Client) *RedisStore {
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

	err = json.Unmarshal([]byte(val), dst)
	if err != nil {
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

	// 使用反射来处理不同的map类型
	mapValue := reflect.ValueOf(dstMap)
	if mapValue.Kind() != reflect.Ptr || mapValue.Elem().Kind() != reflect.Map {
		return fmt.Errorf("dstMap must be a pointer to a map")
	}

	mapElem := mapValue.Elem()
	
	for i, val := range vals {
		if val != nil {
			var result interface{}
			if err := json.Unmarshal([]byte(val.(string)), &result); err != nil {
				return err
			}
			mapElem.SetMapIndex(reflect.ValueOf(keys[i]), reflect.ValueOf(result))
		}
	}

	return nil
}

// Exists 批量检查键存在性
func (r *RedisStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool)
	
	for _, key := range keys {
		exists, err := r.client.Exists(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		result[key] = exists > 0
	}
	
	return result, nil
}

// MSet 批量设置键值对，支持TTL
func (r *RedisStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	pipeline := r.client.Pipeline()
	
	for key, value := range items {
		jsonVal, err := json.Marshal(value)
		if err != nil {
			return err
		}
		
		if ttl > 0 {
			pipeline.SetEX(ctx, key, jsonVal, ttl)
		} else {
			pipeline.Set(ctx, key, jsonVal, 0)
		}
	}
	
	_, err := pipeline.Exec(ctx)
	return err
}

// Del 删除指定键
func (r *RedisStore) Del(ctx context.Context, keys ...string) (int64, error) {
	result, err := r.client.Del(ctx, keys...).Result()
	if err != nil {
		return 0, err
	}
	return result, nil
}