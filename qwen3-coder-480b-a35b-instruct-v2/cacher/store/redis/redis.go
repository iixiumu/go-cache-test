package redis

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore implements the Store interface using Redis
type RedisStore struct {
	client redis.Cmdable
}

// NewRedisStore creates a new RedisStore instance
func NewRedisStore(client redis.Cmdable) *RedisStore {
	return &RedisStore{
		client: client,
	}
}

// Get retrieves a value from Redis and unmarshals it into dst
func (r *RedisStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	result, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// Unmarshal JSON into dst
	if err := json.Unmarshal([]byte(result), dst); err != nil {
		return false, err
	}

	return true, nil
}

// MGet retrieves multiple values from Redis and unmarshals them into dstMap
func (r *RedisStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	result, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return err
	}

	// Create a map to hold the results
	resultMap := make(map[string]interface{})

	// Process results
	for i, key := range keys {
		if result[i] != nil {
			var value interface{}
			if err := json.Unmarshal([]byte(result[i].(string)), &value); err != nil {
				return err
			}
			resultMap[key] = value
		}
	}

	// Convert resultMap to the expected dstMap type using reflection
	return r.convertMap(dstMap, resultMap)
}

// Exists checks the existence of multiple keys in Redis
func (r *RedisStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	// Use pipeline to check existence of each key
	pipe := r.client.Pipeline()
	cmds := make([]*redis.IntCmd, len(keys))

	for i, key := range keys {
		cmds[i] = pipe.Exists(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	// Build result map
	result := make(map[string]bool)
	for i, key := range keys {
		result[key] = cmds[i].Val() > 0
	}

	return result, nil
}

// MSet sets multiple key-value pairs in Redis with optional TTL
func (r *RedisStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	// Convert items to JSON strings
	stringItems := make(map[string]interface{}, len(items))
	for key, value := range items {
		jsonBytes, err := json.Marshal(value)
		if err != nil {
			return err
		}
		stringItems[key] = string(jsonBytes)
	}

	// Use pipeline for atomic operation
	pipe := r.client.TxPipeline()
	pipe.MSet(ctx, stringItems)

	// Set TTL for each key if specified
	if ttl > 0 {
		for key := range items {
			pipe.Expire(ctx, key, ttl)
		}
	}

	_, err := pipe.Exec(ctx)
	return err
}

// Del deletes multiple keys from Redis
func (r *RedisStore) Del(ctx context.Context, keys ...string) (int64, error) {
	return r.client.Del(ctx, keys...).Result()
}

// convertMap converts a map[string]interface{} to the target map type using reflection
func (r *RedisStore) convertMap(dstMap interface{}, srcMap map[string]interface{}) error {
	// Get the reflect.Value of dstMap
	dstMapValue := reflect.ValueOf(dstMap)
	
	// Check if dstMap is a pointer
	if dstMapValue.Kind() != reflect.Ptr {
		return nil // Not a pointer, can't modify
	}
	
	// Get the element that dstMap points to
	dstMapElem := dstMapValue.Elem()
	
	// Check if it's a map
	if dstMapElem.Kind() != reflect.Map {
		return nil // Not a map, can't convert
	}
	
	// Check if the map is nil and initialize it if needed
	if dstMapElem.IsNil() {
		mapType := dstMapElem.Type()
		newMap := reflect.MakeMap(mapType)
		dstMapElem.Set(newMap)
	}
	
	// Get the type of the map's value
	mapValueType := dstMapElem.Type().Elem()
	
	// Iterate through srcMap and convert values
	for key, value := range srcMap {
		// Create a new key value
		keyValue := reflect.ValueOf(key)
		
		// Convert the value to the appropriate type
		valueValue := reflect.ValueOf(value)
		
		// If the types don't match, try to convert
		if valueValue.Type() != mapValueType {
			// Try to convert using json marshal/unmarshal
			jsonBytes, err := json.Marshal(value)
			if err != nil {
				continue
			}
			
			// Create a new instance of the target type
			newValue := reflect.New(mapValueType).Interface()
			if err := json.Unmarshal(jsonBytes, newValue); err != nil {
				continue
			}
			
			// Get the actual value (not pointer)
			newValueElem := reflect.ValueOf(newValue).Elem()
			dstMapElem.SetMapIndex(keyValue, newValueElem)
		} else {
			dstMapElem.SetMapIndex(keyValue, valueValue)
		}
	}
	
	return nil
}