package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// redisStore is a Redis implementation of the Store interface
type redisStore struct {
	client *redis.Client
}

// NewRedisStore creates a new Redis store
func NewRedisStore(client *redis.Client) Store {
	return &redisStore{client: client}
}

// Get 从存储后端获取单个值
func (r *redisStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// For simplicity, we're assuming dst is a *string
	// In a real implementation, you'd need to handle different types with serialization
	if strDst, ok := dst.(*string); ok {
		*strDst = val
		return true, nil
	}

	return false, &InvalidArgumentError{"dst must be a *string for this simple implementation"}
}

// MGet 批量获取值到map中
func (r *redisStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	vals, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return err
	}

	// For simplicity, we're assuming dstMap is a *map[string]string
	// In a real implementation, you'd need to handle different types with serialization
	if strMap, ok := dstMap.(*map[string]string); ok {
		*strMap = make(map[string]string)
		for i, key := range keys {
			if vals[i] != nil {
				(*strMap)[key] = vals[i].(string)
			}
		}
		return nil
	}

	return &InvalidArgumentError{"dstMap must be a *map[string]string for this simple implementation"}
}

// Exists 批量检查键存在性
func (r *redisStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	results := make(map[string]bool)
	for _, key := range keys {
		exists, err := r.client.Exists(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		results[key] = exists > 0
	}
	return results, nil
}

// MSet 批量设置键值对，支持TTL
func (r *redisStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	// Convert map[string]interface{} to map[string]interface{} for Redis
	redisItems := make(map[string]interface{}, len(items))
	for k, v := range items {
		// For simplicity, we're assuming values are strings
		// In a real implementation, you'd need to handle serialization
		if strVal, ok := v.(string); ok {
			redisItems[k] = strVal
		} else {
			return &InvalidArgumentError{"values must be strings for this simple implementation"}
		}
	}

	// Set the values
	err := r.client.MSet(ctx, redisItems).Err()
	if err != nil {
		return err
	}

	// Set TTL for each key if specified
	if ttl > 0 {
		for key := range redisItems {
			err = r.client.Expire(ctx, key, ttl).Err()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Del 删除指定键
func (r *redisStore) Del(ctx context.Context, keys ...string) (int64, error) {
	return r.client.Del(ctx, keys...).Result()
}

func TestRedisStore_Get(t *testing.T) {
	// Start a mini Redis server
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	defer mr.Close()

	// Create a Redis client
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	// Create a Redis store
	store := NewRedisStore(client)

	// Test case 1: Key exists
	t.Run("KeyExists", func(t *testing.T) {
		// Set a value in Redis
		err = client.Set(context.Background(), "key1", "value1", 0).Err()
		assert.NoError(t, err)

		// Get the value
		var result string
		found, err := store.Get(context.Background(), "key1", &result)
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "value1", result)
	})

	// Test case 2: Key does not exist
	t.Run("KeyDoesNotExist", func(t *testing.T) {
		var result string
		found, err := store.Get(context.Background(), "nonexistent", &result)
		assert.NoError(t, err)
		assert.False(t, found)
		assert.Equal(t, "", result)
	})
}

func TestRedisStore_MGet(t *testing.T) {
	// Start a mini Redis server
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	defer mr.Close()

	// Create a Redis client
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	// Create a Redis store
	store := NewRedisStore(client)

	// Set some values in Redis
	err = client.MSet(context.Background(), map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}).Err()
	assert.NoError(t, err)

	// Test case: Get multiple values
	t.Run("GetMultipleValues", func(t *testing.T) {
		result := make(map[string]string)
		err := store.MGet(context.Background(), []string{"key1", "key2", "nonexistent"}, &result)
		assert.NoError(t, err)
		assert.Equal(t, "value1", result["key1"])
		assert.Equal(t, "value2", result["key2"])
		_, exists := result["nonexistent"]
		assert.False(t, exists)
	})
}

func TestRedisStore_Exists(t *testing.T) {
	// Start a mini Redis server
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	defer mr.Close()

	// Create a Redis client
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	// Create a Redis store
	store := NewRedisStore(client)

	// Set a value in Redis
	err = client.Set(context.Background(), "key1", "value1", 0).Err()
	assert.NoError(t, err)

	// Test case: Check existence
	t.Run("CheckExistence", func(t *testing.T) {
		results, err := store.Exists(context.Background(), []string{"key1", "nonexistent"})
		assert.NoError(t, err)
		assert.True(t, results["key1"])
		assert.False(t, results["nonexistent"])
	})
}

func TestRedisStore_MSet(t *testing.T) {
	// Start a mini Redis server
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	defer mr.Close()

	// Create a Redis client
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	// Create a Redis store
	store := NewRedisStore(client)

	// Test case 1: Set values without TTL
	t.Run("SetValuesWithoutTTL", func(t *testing.T) {
		items := map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		}
		err := store.MSet(context.Background(), items, 0)
		assert.NoError(t, err)

		// Verify the values were set
		val1, err := client.Get(context.Background(), "key1").Result()
		assert.NoError(t, err)
		assert.Equal(t, "value1", val1)

		val2, err := client.Get(context.Background(), "key2").Result()
		assert.NoError(t, err)
		assert.Equal(t, "value2", val2)
	})

	// Test case 2: Set values with TTL
	t.Run("SetValuesWithTTL", func(t *testing.T) {
		items := map[string]interface{}{
			"key3": "value3",
		}
		ttl := 100 * time.Millisecond
		err := store.MSet(context.Background(), items, ttl)
		assert.NoError(t, err)

		// Verify the value was set
		val, err := client.Get(context.Background(), "key3").Result()
		assert.NoError(t, err)
		assert.Equal(t, "value3", val)

		// Wait for TTL to expire
		time.Sleep(ttl + 10*time.Millisecond)

		// Verify the value has expired
		_, err = client.Get(context.Background(), "key3").Result()
		assert.Equal(t, redis.Nil, err)
	})
}

func TestRedisStore_Del(t *testing.T) {
	// Start a mini Redis server
	mr, err := miniredis.Run()
	assert.NoError(t, err)
	defer mr.Close()

	// Create a Redis client
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	// Create a Redis store
	store := NewRedisStore(client)

	// Set some values in Redis
	err = client.MSet(context.Background(), map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}).Err()
	assert.NoError(t, err)

	// Test case: Delete keys
	t.Run("DeleteKeys", func(t *testing.T) {
		count, err := store.Del(context.Background(), "key1", "key2", "nonexistent")
		assert.NoError(t, err)
		assert.Equal(t, int64(2), count)

		// Verify the keys were deleted
		_, err = client.Get(context.Background(), "key1").Result()
		assert.Equal(t, redis.Nil, err)

		_, err = client.Get(context.Background(), "key2").Result()
		assert.Equal(t, redis.Nil, err)

		// key3 should still exist
		val, err := client.Get(context.Background(), "key3").Result()
		assert.NoError(t, err)
		assert.Equal(t, "value3", val)
	})
}
