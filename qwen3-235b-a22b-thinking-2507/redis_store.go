package cache

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	redis "github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(client *redis.Client) Store {
	return &RedisStore{client: client}
}

func (r *RedisStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if err := json.Unmarshal([]byte(val), dst); err != nil {
		return false, err
	}
	return true, nil
}

func (r *RedisStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	results, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return err
	}

	dstMapVal := reflect.ValueOf(dstMap).Elem()
	mapType := dstMapVal.Type()
	valueType := mapType.Elem()

	for i, key := range keys {
		if results[i] == nil {
			continue
		}

		data, ok := results[i].(string)
		if !ok {
			continue
		}

		elem := reflect.New(valueType).Interface()
		if err := json.Unmarshal([]byte(data), elem); err != nil {
			return err
		}
		dstMapVal.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(elem).Elem())
	}
	return nil
}

func (r *RedisStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	existsMap := make(map[string]bool)
	for _, key := range keys {
		count, err := r.client.Exists(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		existsMap[key] = count > 0
	}
	return existsMap, nil
}

func (r *RedisStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	pipeline := r.client.Pipeline()
	for key, value := range items {
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		pipeline.Set(ctx, key, data, ttl)
	}
	_, err := pipeline.Exec(ctx)
	return err
}

func (r *RedisStore) Del(ctx context.Context, keys ...string) (int64, error) {
	return r.client.Del(ctx, keys...).Result()
}