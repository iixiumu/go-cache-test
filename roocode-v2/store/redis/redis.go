package redis

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"go-cache/store"

	"github.com/go-redis/redis/v8"
)

// RedisStore 是基于go-redis的Store接口实现
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore 创建一个新的RedisStore实例
func NewRedisStore(client *redis.Client) store.Store {
	return &RedisStore{
		client: client,
	}
}

// Get 从Redis获取单个值
func (r *RedisStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		// 键不存在
		return false, nil
	}
	if err != nil {
		// 其他错误
		return false, err
	}

	// 反序列化值
	if err := json.Unmarshal([]byte(val), dst); err != nil {
		return false, err
	}

	return true, nil
}

// MGet 批量从Redis获取值到map中
func (r *RedisStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	// 执行MGet命令
	result, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return err
	}

	// 使用反射处理dstMap
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr {
		return nil // dstMap必须是指针
	}
	dstMapValue = dstMapValue.Elem()
	if dstMapValue.Kind() != reflect.Map {
		return nil // dstMap必须是指向map的指针
	}

	// 获取map的键和值类型
	mapType := dstMapValue.Type()
	keyType := mapType.Key()
	valueType := mapType.Elem()

	// 创建新的map
	newMap := reflect.MakeMap(mapType)

	// 遍历结果
	for i, val := range result {
		if val == nil {
			// 键不存在，跳过
			continue
		}

		// 将字符串值转换为字节切片
		strVal, ok := val.(string)
		if !ok {
			continue
		}

		// 创建值的实例
		valuePtr := reflect.New(valueType)
		value := valuePtr.Interface()

		// 反序列化
		if err := json.Unmarshal([]byte(strVal), value); err != nil {
			continue
		}

		// 设置map中的值
		mapKey := reflect.ValueOf(keys[i]).Convert(keyType)
		mapValue := reflect.ValueOf(value).Elem()
		newMap.SetMapIndex(mapKey, mapValue)
	}

	// 设置dstMap的值
	dstMapValue.Set(newMap)
	return nil
}

// Exists 批量检查键存在性
func (r *RedisStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	// 使用MGet检查键是否存在
	result, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	// 构建结果
	exists := make(map[string]bool)
	for i, key := range keys {
		exists[key] = result[i] != nil
	}

	return exists, nil
}

// MSet 批量设置键值对，支持TTL
func (r *RedisStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	// 创建管道
	pipe := r.client.TxPipeline()

	// 批量设置
	for key, value := range items {
		// 序列化值
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}

		// 添加到管道
		if ttl > 0 {
			pipe.Set(ctx, key, data, ttl)
		} else {
			pipe.Set(ctx, key, data, 0)
		}
	}

	// 执行管道
	_, err := pipe.Exec(ctx)
	return err
}

// Del 删除指定键
func (r *RedisStore) Del(ctx context.Context, keys ...string) (int64, error) {
	return r.client.Del(ctx, keys...).Result()
}
