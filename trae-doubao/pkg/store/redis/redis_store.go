package redis

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/xiumu/git/me/go-cache/trae/pkg/store"
)

// RedisStore 基于Redis的Store实现
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
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, err
	}

	return true, json.Unmarshal([]byte(val), dst)
}

// MGet 批量从Redis获取值
func (r *RedisStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	// 使用反射检查dstMap是否为*map[string]T类型
	mapValue := reflect.ValueOf(dstMap)
	if mapValue.Kind() != reflect.Ptr || mapValue.Elem().Kind() != reflect.Map {
		return errors.New("dstMap must be a pointer to a map")
	}

	// 调用Redis的MGet命令
	results, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return err
	}

	// 创建结果map
	resultMap := map[string]interface{}{}
	for i, key := range keys {
		if results[i] != nil {
			// 将interface{}转换为字符串，然后反序列化为具体类型
			valStr, ok := results[i].(string)
			if !ok {
				return errors.New("invalid value type from redis")
			}

			// 创建一个临时变量来存储反序列化后的值
			var val interface{}
			if err := json.Unmarshal([]byte(valStr), &val); err != nil {
				return err
			}

			resultMap[key] = val
		}
	}

	// 将结果设置到dstMap
	return setMapValue(dstMap, resultMap)
}

// Exists 批量检查键存在性
func (r *RedisStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	// 使用Pipeline批量检查键存在性
	pipe := r.client.Pipeline()
	defer pipe.Close()

	// 为每个键发送EXISTS命令
	commands := make(map[string]*redis.IntCmd)
	for _, key := range keys {
		commands[key] = pipe.Exists(ctx, key)
	}

	// 执行Pipeline
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, err
	}

	// 收集结果
	result := make(map[string]bool)
	for key, cmd := range commands {
		count, err := cmd.Result()
		if err != nil {
			return nil, err
		}
		result[key] = count > 0
	}

	return result, nil
}

// MSet 批量设置键值对，支持TTL
func (r *RedisStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	// 如果没有项目要设置，直接返回
	if len(items) == 0 {
		return nil
	}

	// 使用Pipeline批量设置键值对
	pipe := r.client.Pipeline()
	defer pipe.Close()

	for key, value := range items {
		// 将值序列化为JSON
		valBytes, err := json.Marshal(value)
		if err != nil {
			return err
		}

		valStr := string(valBytes)

		if ttl > 0 {
			// 设置带TTL的键
			pipe.SetEX(ctx, key, ttl, valStr)
		} else {
			// 设置永不过期的键
			pipe.Set(ctx, key, valStr, 0)
		}
	}

	// 执行Pipeline
	_, err := pipe.Exec(ctx)
	return err
}

// Del 删除指定键
func (r *RedisStore) Del(ctx context.Context, keys ...string) (int64, error) {
	return r.client.Del(ctx, keys...).Result()
}

// 辅助函数：使用反射设置map值
func setMapValue(dstMap interface{}, srcMap map[string]interface{}) error {
	// 获取dstMap的反射值
	dstValue := reflect.ValueOf(dstMap).Elem()

	// 清空dstMap
	dstValue.Set(reflect.MakeMap(dstValue.Type()))

	// 获取map的键类型和值类型
	keyType := dstValue.Type().Key()
	valueType := dstValue.Type().Elem()

	// 遍历srcMap，设置到dstMap
	for k, v := range srcMap {
		// 转换键类型
		keyValue := reflect.ValueOf(k)
		if !keyValue.Type().AssignableTo(keyType) {
			return errors.New("key type mismatch")
		}

		// 转换值类型
		valueValue := reflect.ValueOf(v)
		if !valueValue.Type().AssignableTo(valueType) {
			// 尝试转换类型
			if !valueValue.Type().ConvertibleTo(valueType) {
				return errors.New("value type mismatch and cannot be converted")
			}
			valueValue = valueValue.Convert(valueType)
		}

		// 设置到map
		dstValue.SetMapIndex(keyValue, valueValue)
	}

	return nil
}