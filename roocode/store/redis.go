package store

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/go-redis/redis/v8"
)

// redisStore 是基于Redis的Store实现
type redisStore struct {
	client redis.Cmdable
}

// NewRedisStore 创建一个新的Redis存储实例
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

	// 反序列化到dst
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

	// 创建一个键到值的映射
	keyValueMap := make(map[string]*string)
	for i, key := range keys {
		if result[i] != nil {
			if val, ok := result[i].(string); ok {
				keyValueMap[key] = &val
			}
		}
	}

	// 使用反射将结果设置到dstMap
	// dstMap应该是指向map的指针
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr || dstMapValue.IsNil() {
		return nil
	}

	dstMapElem := dstMapValue.Elem()
	if dstMapElem.Kind() != reflect.Map {
		return nil
	}

	// 获取map的键和值类型
	mapKeyType := dstMapElem.Type().Key()
	mapValueType := dstMapElem.Type().Elem()

	// 填充map
	for key, val := range keyValueMap {
		if val == nil {
			continue
		}

		// 创建map键
		mapKey := reflect.ValueOf(key).Convert(mapKeyType)

		// 创建map值
		mapValue := reflect.New(mapValueType).Interface()
		err := json.Unmarshal([]byte(*val), mapValue)
		if err != nil {
			continue
		}

		// 设置map元素
		dstMapElem.SetMapIndex(mapKey, reflect.ValueOf(mapValue).Elem())
	}

	return nil
}

// Exists 批量检查键在Redis中的存在性
func (r *redisStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool)

	// Redis的Exists命令可以检查多个键
	exists, err := r.client.Exists(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	// 如果所有键都不存在
	if exists == 0 {
		for _, key := range keys {
			result[key] = false
		}
		return result, nil
	}

	// 如果所有键都存在
	if int(exists) == len(keys) {
		for _, key := range keys {
			result[key] = true
		}
		return result, nil
	}

	// 需要逐个检查键的存在性
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
	// 将items转换为map[string]interface{}，其中值是序列化后的字符串
	stringItems := make(map[string]interface{}, len(items))
	for key, value := range items {
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		stringItems[key] = string(data)
	}

	// 执行MSet
	err := r.client.MSet(ctx, stringItems).Err()
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
func (r *redisStore) Del(ctx context.Context, keys ...string) (int64, error) {
	return r.client.Del(ctx, keys...).Result()
}
