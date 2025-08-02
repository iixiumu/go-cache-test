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

// NewRedisStore 创建新的Redis存储实例
func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{
		client: client,
	}
}

// Get 从Redis获取单个值
func (r *RedisStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	// 验证dst是指针
	if reflect.TypeOf(dst).Kind() != reflect.Ptr {
		return false, fmt.Errorf("dst must be a pointer")
	}

	result, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil // 键不存在
		}
		return false, fmt.Errorf("redis get failed: %w", err)
	}

	// 反序列化JSON数据
	if err := json.Unmarshal([]byte(result), dst); err != nil {
		return false, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return true, nil
}

// MGet 批量获取值到map中
func (r *RedisStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	// 验证dstMap是map指针
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr || dstMapValue.Elem().Kind() != reflect.Map {
		return fmt.Errorf("dstMap must be a pointer to map")
	}

	if len(keys) == 0 {
		return nil
	}

	// 批量获取
	results, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return fmt.Errorf("redis mget failed: %w", err)
	}

	mapValue := dstMapValue.Elem()
	mapType := mapValue.Type()
	valueType := mapType.Elem()

	// 确保map已初始化
	if mapValue.IsNil() {
		mapValue.Set(reflect.MakeMap(mapType))
	}

	// 处理结果
	for i, result := range results {
		if result == nil {
			continue // 键不存在
		}

		key := keys[i]
		resultStr, ok := result.(string)
		if !ok {
			return fmt.Errorf("unexpected result type for key %s", key)
		}

		// 创建目标类型的新实例
		valuePtr := reflect.New(valueType)
		if err := json.Unmarshal([]byte(resultStr), valuePtr.Interface()); err != nil {
			return fmt.Errorf("failed to unmarshal JSON for key %s: %w", key, err)
		}

		// 设置到map中
		mapValue.SetMapIndex(reflect.ValueOf(key), valuePtr.Elem())
	}

	return nil
}

// Exists 批量检查键存在性
func (r *RedisStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	if len(keys) == 0 {
		return make(map[string]bool), nil
	}

	result := make(map[string]bool)

	// Redis EXISTS命令可以检查多个键
	existsResult, err := r.client.Exists(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("redis exists failed: %w", err)
	}

	// 如果所有键都不存在
	if existsResult == 0 {
		for _, key := range keys {
			result[key] = false
		}
		return result, nil
	}

	// 需要逐个检查每个键的存在性
	for _, key := range keys {
		exists, err := r.client.Exists(ctx, key).Result()
		if err != nil {
			return nil, fmt.Errorf("redis exists failed for key %s: %w", key, err)
		}
		result[key] = exists > 0
	}

	return result, nil
}

// MSet 批量设置键值对，支持TTL
func (r *RedisStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	// 如果没有TTL，使用MSET
	if ttl <= 0 {
		pairs := make([]interface{}, 0, len(items)*2)
		for key, value := range items {
			jsonData, err := json.Marshal(value)
			if err != nil {
				return fmt.Errorf("failed to marshal JSON for key %s: %w", key, err)
			}
			pairs = append(pairs, key, string(jsonData))
		}

		if err := r.client.MSet(ctx, pairs...).Err(); err != nil {
			return fmt.Errorf("redis mset failed: %w", err)
		}
		return nil
	}

	// 有TTL时，使用pipeline批量设置
	pipe := r.client.Pipeline()
	for key, value := range items {
		jsonData, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON for key %s: %w", key, err)
		}
		pipe.Set(ctx, key, string(jsonData), ttl)
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

	result, err := r.client.Del(ctx, keys...).Result()
	if err != nil {
		return 0, fmt.Errorf("redis del failed: %w", err)
	}

	return result, nil
}
