package redis

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"github.com/redis/go-redis/v9"
)

// Store 实现了store.Store接口，使用Redis作为后端存储
type Store struct {
	client redis.UniversalClient
}

// Options 是Redis存储的配置选项
type Options struct {
	// Redis客户端选项
	RedisOptions *redis.UniversalOptions

	// 如果提供了已存在的Redis客户端，将优先使用它
	Client redis.UniversalClient
}

// New 创建一个新的Redis存储实例
func New(opts Options) (*Store, error) {
	var client redis.UniversalClient

	if opts.Client != nil {
		client = opts.Client
	} else if opts.RedisOptions != nil {
		client = redis.NewUniversalClient(opts.RedisOptions)
	} else {
		return nil, errors.New("either Client or RedisOptions must be provided")
	}

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &Store{
		client: client,
	}, nil
}

// Get 从Redis获取键对应的值，并将其解析到value中
func (s *Store) Get(ctx context.Context, key string, value interface{}) (bool, error) {
	data, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, err
	}

	if err := json.Unmarshal(data, value); err != nil {
		return false, err
	}

	return true, nil
}

// MGet 从Redis批量获取多个键的值，并将结果解析到values中
func (s *Store) MGet(ctx context.Context, keys []string, values interface{}) error {
	// 检查values是否为map类型
	valuesVal := reflect.ValueOf(values)
	if valuesVal.Kind() != reflect.Ptr || valuesVal.Elem().Kind() != reflect.Map {
		return errors.New("values must be a pointer to a map")
	}

	// 获取map的类型信息
	mapVal := valuesVal.Elem()
	mapType := mapVal.Type()
	if mapType.Key().Kind() != reflect.String {
		return errors.New("map key must be string")
	}

	// 如果没有键，直接返回
	if len(keys) == 0 {
		return nil
	}

	// 执行批量获取
	cmd := s.client.MGet(ctx, keys...)
	if cmd.Err() != nil && !errors.Is(cmd.Err(), redis.Nil) {
		return cmd.Err()
	}

	results, err := cmd.Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return err
	}

	// 解析结果到map中
	for i, key := range keys {
		if i >= len(results) || results[i] == nil {
			continue // 跳过不存在的键
		}

		strVal, ok := results[i].(string)
		if !ok {
			continue // 跳过非字符串值
		}

		// 创建一个新的目标值实例
		valType := mapType.Elem()
		newVal := reflect.New(valType)

		// 解析JSON
		if err := json.Unmarshal([]byte(strVal), newVal.Interface()); err != nil {
			continue // 跳过解析错误
		}

		// 设置到map中
		mapVal.SetMapIndex(reflect.ValueOf(key), newVal.Elem())
	}

	return nil
}

// Exists 检查多个键是否存在于Redis中
func (s *Store) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool, len(keys))

	// 初始化所有键为不存在
	for _, key := range keys {
		result[key] = false
	}

	// 如果没有键，直接返回
	if len(keys) == 0 {
		return result, nil
	}

	// 使用pipeline批量检查键是否存在
	pipe := s.client.Pipeline()
	cmds := make(map[string]*redis.IntCmd, len(keys))

	for _, key := range keys {
		cmds[key] = pipe.Exists(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}

	// 处理结果
	for key, cmd := range cmds {
		val, err := cmd.Result()
		if err != nil {
			continue // 跳过错误
		}
		result[key] = val > 0
	}

	return result, nil
}

// MSet 批量设置多个键值对到Redis
func (s *Store) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	// 使用pipeline批量设置
	pipe := s.client.Pipeline()

	for key, value := range items {
		// 序列化值
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}

		// 设置键值对
		if ttl > 0 {
			pipe.Set(ctx, key, data, ttl)
		} else {
			pipe.Set(ctx, key, data, 0)
		}
	}

	// 执行pipeline
	_, err := pipe.Exec(ctx)
	return err
}

// Del 从Redis中删除一个或多个键
func (s *Store) Del(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	// 执行删除操作
	result, err := s.client.Del(ctx, keys...).Result()
	if err != nil {
		return 0, err
	}

	return result, nil
}

// Close 关闭Redis连接
func (s *Store) Close() error {
	return s.client.Close()
}