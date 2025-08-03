package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/redis/go-redis/v9"
	"go-cache/store"
)

// redisStore redis存储
// 实现了store.Store接口
type redisStore struct {
	client redis.UniversalClient
}

// NewRedisStore 创建redis存储
func NewRedisStore(client redis.UniversalClient) store.Store {
	return &redisStore{client: client}
}

// Get 从redis获取单个值
func (s *redisStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	val, err := s.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, s.unmarshal(val, dst)
}

// MGet 批量获取值到map中
func (s *redisStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	if len(keys) == 0 {
		return nil
	}

	vals, err := s.client.MGet(ctx, keys...).Result()
	if err != nil {
		return err
	}

	return s.unmarshalMap(keys, vals, dstMap)
}

// Exists 批量检查键存在性
func (s *redisStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
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

// MSet 批量设置键值对，支持TTL
func (s *redisStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	pipe := s.client.Pipeline()
	for key, value := range items {
		val, err := s.marshal(value)
		if err != nil {
			return err
		}
		pipe.Set(ctx, key, val, ttl)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// Del 删除指定键
func (s *redisStore) Del(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}
	return s.client.Del(ctx, keys...).Result()
}

// marshal 序列化
func (s *redisStore) marshal(value interface{}) ([]byte, error) {
	return json.Marshal(map[string]interface{}{"value": value})
}

// unmarshal 反序列化
func (s *redisStore) unmarshal(data []byte, dst interface{}) error {
	val := reflect.ValueOf(dst)
	if val.Kind() != reflect.Ptr {
		return errors.New("dst must be a pointer")
	}

	temp := make(map[string]interface{})
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	value, ok := temp["value"]
	if !ok {
		return errors.New("invalid data format")
	}

	// 使用json的unmarshal进行类型转换
	jsonData, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return json.Unmarshal(jsonData, dst)
}

// unmarshalMap 批量反序列化到map
func (s *redisStore) unmarshalMap(keys []string, vals []interface{}, dstMap interface{}) error {
	dstVal := reflect.ValueOf(dstMap)
	if dstVal.Kind() != reflect.Ptr || dstVal.Elem().Kind() != reflect.Map {
		return errors.New("dstMap must be a pointer to a map")
	}

	mapVal := dstVal.Elem()
	mapType := mapVal.Type()
	keyType := mapType.Key()
	valType := mapType.Elem()

	if keyType.Kind() != reflect.String {
		return fmt.Errorf("map key type must be string, but got %s", keyType.Kind())
	}

	for i, val := range vals {
		if val == nil {
			continue
		}

		strVal, ok := val.(string)
		if !ok {
			return fmt.Errorf("redis value is not a string: %v", val)
		}

		newVal := reflect.New(valType)
		if err := s.unmarshal([]byte(strVal), newVal.Interface()); err != nil {
			return err
		}

		mapVal.SetMapIndex(reflect.ValueOf(keys[i]), newVal.Elem())
	}

	return nil
}