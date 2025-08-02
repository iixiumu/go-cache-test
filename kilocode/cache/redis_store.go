package cache

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/go-redis/redis/v8"
)

// redisStore 实现了Store接口，使用Redis作为存储后端
type redisStore struct {
	client redis.Cmdable
}

// NewRedisStore 创建一个新的Redis Store实例
func NewRedisStore(client redis.Cmdable) Store {
	return &redisStore{
		client: client,
	}
}

// Get 从Redis获取单个值
func (r *redisStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// 反序列化值到dst
	err = json.Unmarshal([]byte(val), dst)
	if err != nil {
		return false, err
	}

	return true, nil
}

// MGet 批量从Redis获取值
func (r *redisStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	result, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return err
	}

	// 确保dstMap是指向map的指针
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr || dstMapValue.Elem().Kind() != reflect.Map {
		return nil
	}

	// 获取map的元素类型
	dstMapElem := dstMapValue.Elem()
	mapValueType := dstMapElem.Type().Elem()

	// 填充结果map
	for i, key := range keys {
		if result[i] == nil {
			continue
		}

		// 创建对应类型的值
		value := reflect.New(mapValueType).Interface()

		// 反序列化
		valStr, ok := result[i].(string)
		if !ok {
			continue
		}

		err := json.Unmarshal([]byte(valStr), value)
		if err != nil {
			continue
		}

		// 设置map值
		dstMapElem.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value).Elem())
	}

	return nil
}

// Exists 批量检查键在Redis中的存在性
func (r *redisStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool)

	// Redis的EXISTS命令可以检查多个键，但为了简化，我们逐个检查
	for _, key := range keys {
		exists, err := r.client.Exists(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		result[key] = exists > 0
	}

	return result, nil
}

// MSet 批量设置键值对到Redis
func (r *redisStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	// 如果没有设置TTL，使用MSet批量设置
	if ttl <= 0 {
		// 准备要设置的数据
		data := make(map[string]interface{}, len(items))
		for key, value := range items {
			// 序列化值
			valBytes, err := json.Marshal(value)
			if err != nil {
				return err
			}
			data[key] = string(valBytes)
		}

		// 执行MSet
		err := r.client.MSet(ctx, data).Err()
		if err != nil {
			return err
		}
	} else {
		// 如果设置了TTL，逐个设置键值对并设置过期时间
		pipe := r.client.TxPipeline()
		for key, value := range items {
			// 序列化值
			valBytes, err := json.Marshal(value)
			if err != nil {
				return err
			}
			pipe.Set(ctx, key, string(valBytes), ttl)
		}
		_, err := pipe.Exec(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

// Del 从Redis删除指定键
func (r *redisStore) Del(ctx context.Context, keys ...string) (int64, error) {
	return r.client.Del(ctx, keys...).Result()
}
