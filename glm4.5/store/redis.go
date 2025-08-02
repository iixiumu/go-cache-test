package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"
	"go-cache/internal"
)

// RedisStore Redis实现的Store接口
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore 创建Redis存储实例
func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

// Get 从Redis获取单个值
func (r *RedisStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
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

	keyType, valueType, err := internal.GetTypeOfMap(dstMap)
	if err != nil {
		return err
	}

	results, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return err
	}

	for i, key := range keys {
		if results[i] == nil {
			continue
		}

		strValue, ok := results[i].(string)
		if !ok {
			continue
		}

		var value interface{}
		if err := json.Unmarshal([]byte(strValue), &value); err != nil {
			continue
		}

		if err := internal.SetMapValueWithType(dstMap, key, value, keyType, valueType); err != nil {
			return err
		}
	}

	return nil
}

// Exists 批量检查键存在性
func (r *RedisStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	if len(keys) == 0 {
		return make(map[string]bool), nil
	}

	result := make(map[string]bool)
	exists, err := r.client.Exists(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	for i, key := range keys {
		result[key] = i < int(exists)
	}

	return result, nil
}

// MSet 批量设置键值对，支持TTL
func (r *RedisStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	pipe := r.client.Pipeline()
	for key, value := range items {
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}

		if ttl > 0 {
			pipe.Set(ctx, key, data, ttl)
		} else {
			pipe.Set(ctx, key, data, 0)
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

	return r.client.Del(ctx, keys...).Result()
}

// Close 关闭Redis连接
func (r *RedisStore) Close() error {
	return r.client.Close()
}

// RedisStoreOptions Redis存储选项
type RedisStoreOptions struct {
	Addr     string
	Password string
	DB       int
}

// NewRedisStoreWithOptions 使用选项创建Redis存储实例
func NewRedisStoreWithOptions(opts RedisStoreOptions) *RedisStore {
	client := redis.NewClient(&redis.Options{
		Addr:     opts.Addr,
		Password: opts.Password,
		DB:       opts.DB,
	})

	return NewRedisStore(client)
}

// NewTestRedisStore 创建测试用的Redis存储实例
func NewTestRedisStore(addr string) *RedisStore {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	return NewRedisStore(client)
}

// Ping 检查Redis连接
func (r *RedisStore) Ping(ctx context.Context) error {
	_, err := r.client.Ping(ctx).Result()
	return err
}