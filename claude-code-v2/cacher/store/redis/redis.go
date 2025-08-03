package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/redis/go-redis/v9"
	"go-cache/cacher/store"
)

// Store Redis实现的Store接口
type Store struct {
	client redis.Cmdable
}

// NewStore 创建新的Redis Store实例
func NewStore(client redis.Cmdable) *Store {
	return &Store{
		client: client,
	}
}

// Get 从Redis获取单个值
func (s *Store) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	val, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("redis get error: %w", err)
	}

	// 反序列化JSON到目标对象
	if err := json.Unmarshal([]byte(val), dst); err != nil {
		return false, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return true, nil
}

// MGet 批量获取值到map中
func (s *Store) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	if len(keys) == 0 {
		return nil
	}

	// 验证dstMap是map指针
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr || dstMapValue.Elem().Kind() != reflect.Map {
		return fmt.Errorf("dstMap must be a pointer to map")
	}

	mapValue := dstMapValue.Elem()
	mapType := mapValue.Type()
	valueType := mapType.Elem()

	// 如果map为nil，初始化它
	if mapValue.IsNil() {
		mapValue.Set(reflect.MakeMap(mapType))
	}

	// 执行Redis MGET
	vals, err := s.client.MGet(ctx, keys...).Result()
	if err != nil {
		return fmt.Errorf("redis mget error: %w", err)
	}

	// 处理结果
	for i, val := range vals {
		if val == nil {
			continue // 跳过不存在的键
		}

		// 创建值类型的新实例
		valuePtr := reflect.New(valueType)
		
		// 反序列化JSON
		if err := json.Unmarshal([]byte(val.(string)), valuePtr.Interface()); err != nil {
			return fmt.Errorf("failed to unmarshal JSON for key %s: %w", keys[i], err)
		}

		// 设置到map中
		keyValue := reflect.ValueOf(keys[i])
		mapValue.SetMapIndex(keyValue, valuePtr.Elem())
	}

	return nil
}

// Exists 批量检查键存在性
func (s *Store) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool)
	
	if len(keys) == 0 {
		return result, nil
	}

	// 使用pipeline批量检查存在性
	pipe := s.client.Pipeline()
	cmds := make([]*redis.IntCmd, len(keys))
	
	for i, key := range keys {
		cmds[i] = pipe.Exists(ctx, key)
	}
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("redis exists error: %w", err)
	}
	
	// 收集结果
	for i, cmd := range cmds {
		exists, err := cmd.Result()
		if err != nil {
			return nil, fmt.Errorf("redis exists error for key %s: %w", keys[i], err)
		}
		result[keys[i]] = exists > 0
	}

	return result, nil
}

// MSet 批量设置键值对，支持TTL
func (s *Store) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	// 如果没有TTL，使用MSET批量设置
	if ttl == 0 {
		// 准备键值对切片
		args := make([]interface{}, 0, len(items)*2)
		for key, value := range items {
			// 序列化值为JSON
			jsonData, err := json.Marshal(value)
			if err != nil {
				return fmt.Errorf("failed to marshal JSON for key %s: %w", key, err)
			}
			args = append(args, key, string(jsonData))
		}

		err := s.client.MSet(ctx, args...).Err()
		if err != nil {
			return fmt.Errorf("redis mset error: %w", err)
		}
		return nil
	}

	// 有TTL时使用pipeline批量设置
	pipe := s.client.Pipeline()
	
	for key, value := range items {
		// 序列化值为JSON
		jsonData, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON for key %s: %w", key, err)
		}
		pipe.Set(ctx, key, string(jsonData), ttl)
	}
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("redis pipeline set error: %w", err)
	}

	return nil
}

// Del 删除指定键
func (s *Store) Del(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	deletedCount, err := s.client.Del(ctx, keys...).Result()
	if err != nil {
		return 0, fmt.Errorf("redis del error: %w", err)
	}

	return deletedCount, nil
}

// 确保Store实现了store.Store接口
var _ store.Store = (*Store)(nil)
