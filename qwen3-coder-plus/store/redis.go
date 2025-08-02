package store

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/xiumu/go-cache/cache"
)

// redisStore Redis存储实现
type redisStore struct {
	client redis.Cmdable
}

// NewRedisStore 创建一个新的Redis存储实例
func NewRedisStore(client redis.Cmdable) cache.Store {
	return &redisStore{
		client: client,
	}
}

// Get 从存储后端获取单个值
func (r *redisStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		// 键不存在
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// 反序列化数据到dst
	return true, json.Unmarshal([]byte(val), dst)
}

// MGet 批量获取值到map中
func (r *redisStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	// 执行批量获取
	vals, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return err
	}

	// 检查dstMap是否为指针
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr {
		return nil
	}

	// 获取map的实际值
	dstMapValue = dstMapValue.Elem()
	if dstMapValue.Kind() != reflect.Map {
		return nil
	}

	// 获取map的元素类型
	mapElemType := dstMapValue.Type().Elem()

	// 遍历结果
	for i, key := range keys {
		// 检查值是否存在
		if vals[i] == nil {
			continue
		}

		// 转换值
		val, ok := vals[i].(string)
		if !ok {
			continue
		}

		// 创建一个新的元素实例
		elem := reflect.New(mapElemType).Interface()

		// 反序列化数据
		err := json.Unmarshal([]byte(val), elem)
		if err != nil {
			continue
		}

		// 将值设置到map中
		dstMapValue.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(elem).Elem())
	}

	return nil
}

// Exists 批量检查键存在性
func (r *redisStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	// 使用MGET检查键是否存在
	vals, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	result := make(map[string]bool)
	for i, key := range keys {
		result[key] = vals[i] != nil
	}

	return result, nil
}

// MSet 批量设置键值对，支持TTL
func (r *redisStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	// 创建管道
	pipe := r.client.TxPipeline()

	// 批量设置键值对
	for key, value := range items {
		// 序列化值
		data, err := json.Marshal(value)
		if err != nil {
			continue
		}

		// 设置键值
		pipe.Set(ctx, key, data, ttl)
	}

	// 执行管道命令
	_, err := pipe.Exec(ctx)
	return err
}

// Del 删除指定键
func (r *redisStore) Del(ctx context.Context, keys ...string) (int64, error) {
	return r.client.Del(ctx, keys...).Result()
}