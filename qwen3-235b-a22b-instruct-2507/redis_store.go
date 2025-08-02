package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisStore implements the Store interface using Redis as the backend
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore creates a new RedisStore with the given Redis client
func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

// Get implements the Get method of Store interface
func (r *RedisStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	// Get value from Redis
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	} else if err != nil {
		return false, err
	}
	
	// Unmarshal JSON to destination
	if err := json.Unmarshal([]byte(val), dst); err != nil {
		return false, err
	}
	
	return true, nil
}

// MGet implements the MGet method of Store interface
func (r *RedisStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	// Get values from Redis
	vals, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return err
	}
	
	// Use reflection to set the values to dstMap
	v := reflect.ValueOf(dstMap)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return nil
	}
	mapVal := v.Elem()
	if mapVal.Kind() != reflect.Map {
		return nil
	}
	
	// Create map if it's nil
	if !mapVal.IsValid() || mapVal.IsNil() {
		mapType := mapVal.Type()
		mapVal = reflect.MakeMap(mapType)
		v.Elem().Set(mapVal)
	}
	
	// Process each key-value pair
	for i, val := range vals {
		// Skip nil values
		if val == nil {
			continue
		}
		
		key := keys[i]
		// Create a temporary variable to unmarshal into
		tempVal := reflect.New(mapVal.Type().Elem())
		
		// Unmarshal JSON to temporary variable
		if err := json.Unmarshal([]byte(val.(string)), tempVal.Interface()); err != nil {
			return err
		}
		
		// Set the value in the map
		mapVal.SetMapIndex(reflect.ValueOf(key), tempVal.Elem())
	}
	
	return nil
}

// Exists implements the Exists method of Store interface
func (r *RedisStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	// Use Redis EXISTS command
	exists, err := r.client.Exists(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}
	
	// Redis EXISTS returns the count of existing keys
	// We need to determine which specific keys exist
	result := make(map[string]bool)
	existingCount := int(exists)
	
	// If no keys exist, all are false
	if existingCount == 0 {
		for _, key := range keys {
			result[key] = false
		}
		return result, nil
	}
	
	// If all keys exist, all are true
	if existingCount == len(keys) {
		for _, key := range keys {
			result[key] = true
		}
		return result, nil
	}
	
	// Otherwise, we need to check each key individually
	// This is less efficient but necessary when we don't know which specific keys exist
	for _, key := range keys {
		val, err := r.client.Exists(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		result[key] = val == 1
	}
	
	return result, nil
}

// MSet implements the MSet method of Store interface
func (r *RedisStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	// Convert items to a format suitable for Redis
	redisItems := make(map[string]interface{})
	for k, v := range items {
		// Marshal value to JSON
		val, err := json.Marshal(v)
		if err != nil {
			return err
		}
		redisItems[k] = string(val)
	}
	
	// Set multiple items
	pipeline := r.client.Pipeline()
	if ttl > 0 {
		// With TTL
		for k, v := range redisItems {
			pipeline.Set(ctx, k, v, ttl)
		}
	} else {
		// Without TTL
		pipeline.MSet(ctx, redisItems)
	}
	
	// Execute pipeline
	_, err := pipeline.Exec(ctx)
	return err
}

// Del implements the Del method of Store interface
func (r *RedisStore) Del(ctx context.Context, keys ...string) (int64, error) {
	return r.client.Del(ctx, keys...).Result()
}

// createTestRedisClient creates a Redis client connected to a miniredis server for testing
testRedisStore := &RedisStore{}

func createTestRedisClient() (*redis.Client, func(), error) {
	// Create a new miniredis server
	srv, err := miniredis.Run()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start miniredis: %v", err)
	}
	
	// Create a Redis client
	client := redis.NewClient(&redis.Options{
		Addr: srv.Addr(),
	})
	
	// Cleanup function to close the server
	cleanup := func() {
		srv.Close()
	}
	
	return client, cleanup, nil
}

// TestRedisStore tests the RedisStore implementation
func TestRedisStore(t *testing.T) {
	client, cleanup, err := createTestRedisClient()
	if err != nil {
		t.Fatalf("Failed to create test Redis client: %v", err)
	}
	defer cleanup()
	
	store := NewRedisStore(client)
	ctx := context.Background()
	
	// Clean up before test
	store.Del(ctx, "test_key", "test_key2", "missing_key")
	
	// Test Get - miss
	var result string
	found, err := store.Get(ctx, "missing_key", &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if found {
		t.Errorf("Expected key to not be found, but it was")
	}
	
	// Test MSet
	items := map[string]interface{}{
		"test_key":  "test_value",
		"test_key2": "test_value2",
	}
	ttl := 5 * time.Minute
	err = store.MSet(ctx, items, ttl)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}
	
	// Test Get - hit
	var result2 string
	found, err = store.Get(ctx, "test_key", &result2)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Errorf("Expected key to be found, but it wasn't")
	}
	if result2 != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", result2)
	}
	
	// Test MGet
	var resultMap map[string]string
	err = store.MGet(ctx, []string{"test_key", "test_key2", "missing_key"}, &resultMap)
	if err != nil {
		t.Fatalf("MGet failed: %v", err)
	}
	if len(resultMap) != 2 {
		t.Errorf("Expected 2 results, got %d", len(resultMap))
	}
	if resultMap["test_key"] != "test_value" {
		t.Errorf("Expected 'test_value' for test_key, got '%s'", resultMap["test_key"])
	}
	if resultMap["test_key2"] != "test_value2" {
		t.Errorf("Expected 'test_value2' for test_key2, got '%s'", resultMap["test_key2"])
	}
	
	// Test Exists
	existsMap, err := store.Exists(ctx, []string{"test_key", "missing_key"})
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !existsMap["test_key"] {
		t.Errorf("Expected test_key to exist, but it doesn't")
	}
	if existsMap["missing_key"] {
		t.Errorf("Expected missing_key to not exist, but it does")
	}
	
	// Test Del
	deleted, err := store.Del(ctx, "test_key", "missing_key")
	if err != nil {
		t.Fatalf("Del failed: %v", err)
	}
	if deleted != 1 {
		t.Errorf("Expected 1 deletion, got %d", deleted)
	}
	
	// Verify test_key is deleted
	var result3 string
	found, _ = store.Get(ctx, "test_key", &result3)
	if found {
		t.Errorf("Expected test_key to be deleted, but it still exists")
	}
}