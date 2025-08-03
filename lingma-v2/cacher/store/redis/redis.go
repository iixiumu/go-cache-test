package redis

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"github.com/redis/go-redis/v9"
	"go-cache/cacher/store"
)

// redisStore 是 Store 接口的 Redis 实现
type redisStore struct {
	client redis.Cmdable
}

// NewRedisStore 创建一个新的 Redis 存储实例
func NewRedisStore(client redis.Cmdable) store.Store {
	return &redisStore{
		client: client,
	}
}

// Get 从 Redis 获取单个值
func (r *redisStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	result, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// 反序列化 JSON 到目标对象
	if err := json.Unmarshal([]byte(result), dst); err != nil {
		return false, err
	}

	return true, nil
}

// MGet 批量从 Redis 获取值
func (r *redisStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	result, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return err
	}

	// 使用反射处理目标 map
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr || dstMapValue.Elem().Kind() != reflect.Map {
		return errors.New("dstMap must be a pointer to a map")
	}

	mapValue := dstMapValue.Elem()
	mapValue.Set(reflect.MakeMap(mapValue.Type()))

	keyType := mapValue.Type().Key()
	elemType := mapValue.Type().Elem()

	for i, key := range keys {
		if result[i] == nil {
			continue
		}

		// 创建 map 元素
		elem := reflect.New(elemType).Interface()

		// 反序列化 JSON
		if str, ok := result[i].(string); ok {
			if err := json.Unmarshal([]byte(str), elem); err != nil {
				return err
			}

			// 设置 map 值
			mapValue.SetMapIndex(reflect.ValueOf(key).Convert(keyType), reflect.ValueOf(elem).Elem())
		}
	}

	return nil
}

// Exists 检查键在 Redis 中是否存在
func (r *redisStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	// 使用单独的 EXISTS 命令检查每个键
	results := make(map[string]bool)
	for _, key := range keys {
		exists, err := r.client.Exists(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		results[key] = exists > 0
	}

	return results, nil
}

// MSet 批量设置键值对到 Redis
func (r *redisStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	// 将值序列化为 JSON
	serializedItems := make(map[string]interface{}, len(items))
	for key, value := range items {
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		serializedItems[key] = string(data)
	}

	// 执行 MSet
	if err := r.client.MSet(ctx, serializedItems).Err(); err != nil {
		return err
	}

	// 如果设置了 TTL，则为每个键设置过期时间
	if ttl > 0 {
		for key := range items {
			if err := r.client.Expire(ctx, key, ttl).Err(); err != nil {
				return err
			}
		}
	}

	return nil
}

// Del 从 Redis 删除指定键
func (r *redisStore) Del(ctx context.Context, keys ...string) (int64, error) {
	return r.client.Del(ctx, keys...).Result()
}