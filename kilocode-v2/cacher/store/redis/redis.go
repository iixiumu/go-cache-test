package redis

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"go-cache/cacher/store"

	"github.com/redis/go-redis/v9"
)

// RedisStore 实现Store接口的Redis存储
type RedisStore struct {
	client redis.Cmdable
}

// NewRedisStore 创建一个新的RedisStore实例
func NewRedisStore(client redis.Cmdable) store.Store {
	return &RedisStore{
		client: client,
	}
}

// getMapPtrValue 获取map指针的reflect.Value
func getMapPtrValue(mapPtr interface{}) (reflect.Value, error) {
	v := reflect.ValueOf(mapPtr)
	if v.Kind() != reflect.Ptr {
		return reflect.Value{}, &reflect.ValueError{Method: "getMapPtrValue", Kind: v.Kind()}
	}
	if v.Elem().Kind() != reflect.Map {
		return reflect.Value{}, &reflect.ValueError{Method: "getMapPtrValue", Kind: v.Elem().Kind()}
	}
	return v.Elem(), nil
}

// Get 从Redis获取单个值
func (r *RedisStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		// 键不存在
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// 反序列化JSON到目标变量
	if err := json.Unmarshal([]byte(val), dst); err != nil {
		return false, err
	}

	return true, nil
}

// MGet 批量从Redis获取值
func (r *RedisStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	result, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return err
	}

	// 使用反射来设置目标map
	dstValue, err := getMapPtrValue(dstMap)
	if err != nil {
		return err
	}

	// 清空目标map
	dstValue.Set(reflect.MakeMap(dstValue.Type()))

	// 填充结果
	for i, key := range keys {
		if result[i] == nil {
			// 键不存在，跳过
			continue
		}

		// 创建新元素
		elemType := dstValue.Type().Elem()
		elem := reflect.New(elemType).Interface()

		// 反序列化JSON
		if str, ok := result[i].(string); ok {
			if err := json.Unmarshal([]byte(str), elem); err != nil {
				return err
			}

			// 设置map值
			dstValue.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(elem).Elem())
		}
	}

	return nil
}

// Exists 批量检查键在Redis中的存在性
func (r *RedisStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool)

	// 如果所有键都存在
	exists, err := r.client.Exists(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	// 如果所有键都存在
	if int(exists) == len(keys) {
		for _, key := range keys {
			result[key] = true
		}
		return result, nil
	}

	// 逐个检查键的存在性
	for _, key := range keys {
		exists, err := r.client.Exists(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		result[key] = exists == 1
	}

	return result, nil
}

// MSet 批量设置键值对到Redis
func (r *RedisStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	// 准备数据
	data := make(map[string]interface{}, len(items))
	for key, value := range items {
		// 序列化为JSON
		jsonData, err := json.Marshal(value)
		if err != nil {
			return err
		}
		data[key] = string(jsonData)
	}

	// 执行MSet
	if err := r.client.MSet(ctx, data).Err(); err != nil {
		return err
	}

	// 如果设置了TTL，为每个键设置过期时间
	if ttl > 0 {
		for key := range items {
			if err := r.client.Expire(ctx, key, ttl).Err(); err != nil {
				return err
			}
		}
	}

	return nil
}

// Del 从Redis删除指定键
func (r *RedisStore) Del(ctx context.Context, keys ...string) (int64, error) {
	return r.client.Del(ctx, keys...).Result()
}
