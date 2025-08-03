package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore is a store implementation for Redis.
// It uses json for serialization.
type RedisStore struct {
	client redis.Cmdable
}

// NewRedisStore creates a new RedisStore.
func NewRedisStore(client redis.Cmdable) *RedisStore {
	return &RedisStore{client: client}
}

// Get retrieves a value from redis and unmarshals it into dst.
func (s *RedisStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	if dst == nil {
		return false, errors.New("dst must not be nil")
	}
	if reflect.ValueOf(dst).Kind() != reflect.Ptr {
		return false, errors.New("dst must be a pointer")
	}

	val, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}

	err = json.Unmarshal([]byte(val), dst)
	if err != nil {
		return false, err
	}

	return true, nil
}

// MGet retrieves multiple values from redis.
func (s *RedisStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	if dstMap == nil {
		return errors.New("dstMap must not be nil")
	}
	dstVal := reflect.ValueOf(dstMap)
	if dstVal.Kind() != reflect.Ptr || dstVal.Elem().Kind() != reflect.Map {
		return errors.New("dstMap must be a pointer to a map")
	}

	if len(keys) == 0 {
		return nil
	}

	vals, err := s.client.MGet(ctx, keys...).Result()
	if err != nil {
		return err
	}

	mapVal := dstVal.Elem()
	mapType := mapVal.Type()
	valType := mapType.Elem()

	for i, v := range vals {
		if v == nil {
			continue
		}

		valStr, ok := v.(string)
		if !ok {
			return fmt.Errorf("unexpected type for key %s: %T", keys[i], v)
		}

		newVal := reflect.New(valType)
		err := json.Unmarshal([]byte(valStr), newVal.Interface())
		if err != nil {
			// Try to unmarshal to a pointer if the element type is a pointer
			if valType.Kind() == reflect.Ptr {
				newVal = reflect.New(valType.Elem())
				err = json.Unmarshal([]byte(valStr), newVal.Interface())
				if err != nil {
					return fmt.Errorf("failed to unmarshal value for key %s: %w", keys[i], err)
				}
				mapVal.SetMapIndex(reflect.ValueOf(keys[i]), newVal)
			} else {
				return fmt.Errorf("failed to unmarshal value for key %s: %w", keys[i], err)
			}
		} else {
			mapVal.SetMapIndex(reflect.ValueOf(keys[i]), newVal.Elem())
		}
	}

	return nil
}

// Exists checks if keys exist in redis.
func (s *RedisStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	if len(keys) == 0 {
		return make(map[string]bool), nil
	}

	pipe := s.client.Pipeline()
	cmds := make(map[string]*redis.IntCmd)
	for _, key := range keys {
		cmds[key] = pipe.Exists(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	existsMap := make(map[string]bool)
	for key, cmd := range cmds {
		val, err := cmd.Result()
		if err != nil {
			return nil, err
		}
		existsMap[key] = val > 0
	}

	return existsMap, nil
}

// MSet sets multiple key-value pairs in redis.
func (s *RedisStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	pipe := s.client.Pipeline()
	for key, val := range items {
		jsonVal, err := json.Marshal(val)
		if err != nil {
			return err
		}
		pipe.Set(ctx, key, jsonVal, ttl)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// Del deletes keys from redis.
func (s *RedisStore) Del(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}
	return s.client.Del(ctx, keys...).Result()
}