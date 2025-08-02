package cache

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore 基于Redis的Store实现
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore 创建新的Redis存储实例
func NewRedisStore(client *redis.Client) Store {
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
		return false, err
	}

	// 尝试JSON反序列化
	if err := json.Unmarshal([]byte(result), dst); err != nil {
		return false, err
	}

	return true, nil
}

// MGet 批量获取值
func (r *RedisStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	if len(keys) == 0 {
		return nil
	}

	// 从Redis批量获取
	results, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return err
	}

	// 使用反射设置结果到目标map
	dstValue := reflect.ValueOf(dstMap)
	if dstValue.Kind() != reflect.Ptr || dstValue.Elem().Kind() != reflect.Map {
		return ErrInvalidDstMap
	}

	dstMapValue := dstValue.Elem()
	dstMapType := dstMapValue.Type()

	// 确保map已初始化
	if dstMapValue.IsNil() {
		dstMapValue.Set(reflect.MakeMap(dstMapType))
	}

	// 设置结果
	for i, key := range keys {
		if results[i] != nil {
			// 将Redis结果转换为JSON字符串，然后反序列化
			if jsonStr, ok := results[i].(string); ok {
				// 创建目标类型的零值
				elemType := dstMapType.Elem()
				elemValue := reflect.New(elemType).Elem()

				// 尝试JSON反序列化
				if err := json.Unmarshal([]byte(jsonStr), elemValue.Addr().Interface()); err == nil {
					dstMapValue.SetMapIndex(reflect.ValueOf(key), elemValue)
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

	// 使用Redis的EXISTS命令批量检查
	pipe := r.client.Pipeline()
	cmds := make([]*redis.IntCmd, len(keys))

	for i, key := range keys {
		cmds[i] = pipe.Exists(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	// 构建结果
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

	// 将值序列化为JSON字符串
	jsonItems := make(map[string]interface{})
	for key, value := range items {
		jsonBytes, err := json.Marshal(value)
		if err != nil {
			return err
		}
		jsonItems[key] = string(jsonBytes)
	}

	// 批量设置
	pipe := r.client.Pipeline()
	pipe.MSet(ctx, jsonItems)

	// 如果设置了TTL，为每个键设置过期时间
	if ttl > 0 {
		for key := range items {
			pipe.Expire(ctx, key, ttl)
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

	result, err := r.client.Del(ctx, keys...).Result()
	return result, err
}
