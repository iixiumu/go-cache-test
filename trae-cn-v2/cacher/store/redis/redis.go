package redis

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore 实现了Store接口的Redis存储后端
type RedisStore struct {
	client *redis.Client
}

// GetTTL 获取键的剩余生存时间（仅Redis实现）
func (r *RedisStore) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	return ttl, nil
}

// NewRedisStore 创建一个新的RedisStore实例
// addr: Redis服务器地址
func NewRedisStore(addr string) *RedisStore {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	return &RedisStore{
		client: client,
	}
}

// NewRedisStoreWithClient 使用提供的客户端创建RedisStore实例
func NewRedisStoreWithClient(client *redis.Client) *RedisStore {
	return &RedisStore{
		client: client,
	}
}

// Close 关闭Redis客户端连接
func (r *RedisStore) Close() error {
	return r.client.Close()
}

// Get 从Redis获取单个值
func (r *RedisStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return err
	}

	// 创建目标类型的值
	dstVal := reflect.ValueOf(dst)
	if dstVal.Kind() != reflect.Ptr {
		return false, errors.New("dst must be a pointer")
	}

	dstElem := dstVal.Elem()

	// 反序列化到临时变量
	var temp interface{}
	if err := json.Unmarshal([]byte(val), &temp); err != nil {
		return false, err
	}

	// 根据目标类型进行转换
	if dstElem.Kind() == reflect.Int {
		if floatVal, ok := temp.(float64); ok {
			dstElem.SetInt(int64(floatVal))
		} else if intVal, ok := temp.(int); ok {
			dstElem.SetInt(int64(intVal))
		} else {
			return false, errors.New("cannot convert to int")
		}
	} else if dstElem.Kind() == reflect.Slice {
		if sliceVal, ok := temp.([]interface{}); ok {
			// 创建目标类型的切片
			destSlice := reflect.MakeSlice(dstElem.Type(), len(sliceVal), len(sliceVal))
			for i, item := range sliceVal {
				// 尝试转换每个元素
				if dstElem.Type().Elem().Kind() == reflect.Int {
					if floatVal, ok := item.(float64); ok {
						destSlice.Index(i).SetInt(int64(floatVal))
					} else if intVal, ok := item.(int); ok {
						destSlice.Index(i).SetInt(int64(intVal))
					} else {
						return errors.New("cannot convert slice element to int")
					}
				} else {
					itemVal := reflect.ValueOf(item)
					if itemVal.Type().AssignableTo(dstElem.Type().Elem()) {
						destSlice.Index(i).Set(itemVal)
					} else {
						// 尝试递归转换复杂类型
						newItem := reflect.New(dstElem.Type().Elem()).Interface()
						itemJSON, err := json.Marshal(item)
						if err != nil {
							return errors.New("cannot marshal slice element")
						}
						if err := json.Unmarshal(itemJSON, newItem); err != nil {
									return errors.New("cannot convert slice element")
								}
					destSlice.Index(i).Set(reflect.ValueOf(newItem).Elem())
				}
			}
		}
		dstElem.Set(destSlice)
	} else {
		return false, errors.New("cannot convert to slice")
	}
		// 对于其他类型，尝试直接设置或反序列化
		if reflect.TypeOf(temp).AssignableTo(dstElem.Type()) {
			dstElem.Set(reflect.ValueOf(temp))
		} else {
			// 尝试通过JSON进行转换
			newVal, err := json.Marshal(temp)
			if err != nil {
				return false, err
			}
			if err := json.Unmarshal(newVal, dst); err != nil {
				return false, err
			}
		}
	}

	return true, nil
}

// MGet 批量获取值到map中
func (r *RedisStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	// 使用反射确保dstMap是*map[string]T类型
	mapVal := reflect.ValueOf(dstMap)
	if mapVal.Kind() != reflect.Ptr || mapVal.Elem().Kind() != reflect.Map {
		return errors.New("dstMap must be a pointer to a map")
	}

	// 调用Redis的MGet命令
	values, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return err
	}

	// 确保目标map已初始化
	if mapVal.Elem().IsNil() {
		mapVal.Elem().Set(reflect.MakeMap(mapVal.Elem().Type()))
	}

	// 获取map的值类型
	valueType := mapVal.Elem().Type().Elem()

	// 填充结果map
	for i, key := range keys {
		val := values[i]
		if val != nil {
			// 创建目标类型的值
			destValue := reflect.New(valueType).Interface()
			destValueVal := reflect.ValueOf(destValue).Elem()

			// 反序列化到临时变量
			jsonStr := val.(string)
			var temp interface{}
			if err := json.Unmarshal([]byte(jsonStr), &temp); err != nil {
				return err
			}

			// 根据目标类型进行转换
			if valueType.Kind() == reflect.Int {
				if floatVal, ok := temp.(float64); ok {
					destValueVal.SetInt(int64(floatVal))
				} else if intVal, ok := temp.(int); ok {
					destValueVal.SetInt(int64(intVal))
				} else {
					return errors.New("cannot convert to int")
				}
			} else if valueType.Kind() == reflect.Slice {
				if sliceVal, ok := temp.([]interface{}); ok {
					// 创建目标类型的切片
					destSlice := reflect.MakeSlice(valueType, len(sliceVal), len(sliceVal))
					for i, item := range sliceVal {
						// 尝试转换每个元素
						if valueType.Elem().Kind() == reflect.Int {
							if floatVal, ok := item.(float64); ok {
								destSlice.Index(i).SetInt(int64(floatVal))
							} else if intVal, ok := item.(int); ok {
								destSlice.Index(i).SetInt(int64(intVal))
							} else {
								return false, errors.New("cannot convert slice element to int")
							}
						} else {
							itemVal := reflect.ValueOf(item)
							if itemVal.Type().AssignableTo(valueType.Elem()) {
								destSlice.Index(i).Set(itemVal)
							} else {
								// 尝试递归转换复杂类型
								newItem := reflect.New(valueType.Elem()).Interface()
								itemJSON, err := json.Marshal(item)
								if err != nil {
									return errors.New("cannot marshal slice element")
								}
								if err := json.Unmarshal(itemJSON, newItem); err != nil {
									return errors.New("cannot convert slice element")
								}
								destSlice.Index(i).Set(reflect.ValueOf(newItem).Elem())
							}
						}
					}
					destValueVal.Set(destSlice)
				} else {
					return errors.New("cannot convert to slice")
				}
			} else {
				// 对于其他类型，尝试直接设置或反序列化
				if reflect.TypeOf(temp).AssignableTo(valueType) {
					destValueVal.Set(reflect.ValueOf(temp))
				} else {
					// 尝试通过JSON进行转换
					newVal, err := json.Marshal(temp)
					if err != nil {
						return false, err
					}
					if err := json.Unmarshal(newVal, destValue); err != nil {
						return false, err
					}
				}
			}

			// 设置值到目标map
			mapVal.Elem().SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(destValue).Elem())
		}
	}

	return nil
}

// Exists 批量检查键存在性
func (r *RedisStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	// 创建结果map
	result := make(map[string]bool, len(keys))

	// 使用Pipeline来获取每个键的存在性
	pipe := r.client.Pipeline()
	cmds := make([]*redis.IntCmd, len(keys))
	for i, key := range keys {
		cmds[i] = pipe.Exists(ctx, key)
	}

	// 执行Pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	// 收集结果
	for i, key := range keys {
		result[key] = cmds[i].Val() > 0
	}

	return result, nil
}

// MSet 批量设置键值对，支持TTL
func (r *RedisStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	// 使用Pipeline批量设置
	pipe := r.client.Pipeline()

	for key, value := range items {
		// 序列化值为JSON
		jsonData, err := json.Marshal(value)
		if err != nil {
			return err
		}

		if ttl > 0 {
			pipe.SetEx(ctx, key, jsonData, ttl)
		} else {
			pipe.Set(ctx, key, jsonData, 0)
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