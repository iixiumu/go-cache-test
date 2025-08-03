package redis

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore 是基于 Redis 的存储实现
type RedisStore struct {
	client redis.Cmdable
}

// NewRedisStore 创建一个新的 RedisStore 实例
func NewRedisStore(client redis.Cmdable) *RedisStore {
	return &RedisStore{
		client: client,
	}
}

// Get 从存储后端获取单个值
func (r *RedisStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// 反序列化 JSON 到 dst
	return true, json.Unmarshal([]byte(val), dst)
}

// MGet 批量获取值到map中
func (r *RedisStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	// 验证 dstMap 是一个指向 map 的指针
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr || dstMapValue.Elem().Kind() != reflect.Map {
		return &InvalidTypeError{Message: "dstMap must be a pointer to a map"}
	}

	// 执行 MGet 命令
	result, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return err
	}

	// 初始化目标 map
	dstMapValue.Elem().Set(reflect.MakeMap(dstMapValue.Elem().Type()))

	// 获取 map 的键和值类型
	mapType := dstMapValue.Elem().Type()
	keyType := mapType.Key()
	valueType := mapType.Elem()

	// 填充结果
	for i, key := range keys {
		if result[i] == nil {
			continue
		}

		// 创建 map 的键
		mapKey := reflect.ValueOf(key).Convert(keyType)

		// 创建 map 的值
		mapValuePtr := reflect.New(valueType)
		// 反序列化 JSON 到值
		err = json.Unmarshal([]byte(result[i].(string)), mapValuePtr.Interface())
		if err != nil {
			return err
		}

		// 将值设置到 map 中
		dstMapValue.Elem().SetMapIndex(mapKey, mapValuePtr.Elem())
	}

	return nil
}

// Exists 批量检查键存在性
func (r *RedisStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	// 执行 EXISTS 命令
	result, err := r.client.Exists(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	// 创建结果 map
	exists := make(map[string]bool)

	// 如果所有键都存在
	if int(result) == len(keys) {
		for _, key := range keys {
			exists[key] = true
		}
		return exists, nil
	}

	// 逐个检查键的存在性
	for _, key := range keys {
		count, err := r.client.Exists(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		exists[key] = count > 0
	}

	return exists, nil
}

// MSet 批量设置键值对，支持TTL
func (r *RedisStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	// 使用管道批量设置
	pipe := r.client.TxPipeline()

	// 序列化并设置每个键值对
	for key, value := range items {
		// 序列化值为 JSON
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}

		// 设置键值对
		pipe.Set(ctx, key, data, ttl)
	}

	// 执行管道命令
	_, err := pipe.Exec(ctx)
	return err
}

// Del 删除指定键
func (r *RedisStore) Del(ctx context.Context, keys ...string) (int64, error) {
	return r.client.Del(ctx, keys...).Result()
}

// InvalidTypeError 无效类型错误
type InvalidTypeError struct {
	Message string
}

func (e *InvalidTypeError) Error() string {
	return e.Message
}
