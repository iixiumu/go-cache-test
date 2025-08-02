package cache

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisStore Redis存储实现
type RedisStore struct {
	client redis.Cmdable
}

// NewRedisStore 创建Redis存储实例
func NewRedisStore(client redis.Cmdable) Store {
	return &RedisStore{
		client: client,
	}
}

// Get 从Redis获取单个值
func (r *RedisStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}

	if err := json.Unmarshal(data, dst); err != nil {
		return false, err
	}

	return true, nil
}

// MGet 批量获取值到map中
func (r *RedisStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	if len(keys) == 0 {
		return nil
	}

	// 验证dstMap是指向map的指针
	dstValue := reflect.ValueOf(dstMap)
	if dstValue.Kind() != reflect.Ptr || dstValue.Elem().Kind() != reflect.Map {
		return ErrInvalidMapType
	}

	mapValue := dstValue.Elem()
	mapType := mapValue.Type()
	valueType := mapType.Elem()

	// 批量获取
	values, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return err
	}

	// 处理结果
	for i, value := range values {
		if value == nil {
			continue
		}

		key := keys[i]
		
		// 创建目标类型的新实例
		newValue := reflect.New(valueType)
		
		// 反序列化
		if err := json.Unmarshal([]byte(value.(string)), newValue.Interface()); err != nil {
			continue // 跳过反序列化失败的项
		}

		// 设置到map中
		mapValue.SetMapIndex(reflect.ValueOf(key), newValue.Elem())
	}

	return nil
}

// Exists 批量检查键存在性
func (r *RedisStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	if len(keys) == 0 {
		return make(map[string]bool), nil
	}

	result := make(map[string]bool)
	
	// Redis的EXISTS命令返回存在的键数量，我们需要逐个检查
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
	if len(items) == 0 {
		return nil
	}

	// 序列化所有值
	serializedItems := make(map[string]interface{})
	for key, value := range items {
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		serializedItems[key] = string(data)
	}

	// 批量设置
	if err := r.client.MSet(ctx, serializedItems).Err(); err != nil {
		return err
	}

	// 如果有TTL，需要为每个键设置过期时间
	if ttl > 0 {
		pipe := r.client.Pipeline()
		for key := range items {
			pipe.Expire(ctx, key, ttl)
		}
		_, err := pipe.Exec(ctx)
		return err
	}

	return nil
}

// Del 删除指定键
func (r *RedisStore) Del(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}
	return r.client.Del(ctx, keys...).Result()
}