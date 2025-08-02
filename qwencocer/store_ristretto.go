package cache

import (
	"context"
	"testing"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/stretchr/testify/assert"
)

// ristrettoStore is a Ristretto implementation of the Store interface
type ristrettoStore struct {
	cache *ristretto.Cache
}

// NewRistrettoStore creates a new Ristretto store
func NewRistrettoStore(cache *ristretto.Cache) Store {
	return &ristrettoStore{cache: cache}
}

// Get 从存储后端获取单个值
func (r *ristrettoStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	val, found := r.cache.Get(key)
	if !found {
		return false, nil
	}

	// For simplicity, we're assuming dst is a *string
	// In a real implementation, you'd need to handle different types with reflection
	if strDst, ok := dst.(*string); ok {
		if strVal, ok := val.(string); ok {
			*strDst = strVal
			return true, nil
		}
		return false, &InvalidArgumentError{"cached value is not a string"}
	}

	return false, &InvalidArgumentError{"dst must be a *string for this simple implementation"}
}

// MGet 批量获取值到map中
func (r *ristrettoStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	// For simplicity, we're assuming dstMap is a *map[string]string
	// In a real implementation, you'd need to handle different types with reflection
	if strMap, ok := dstMap.(*map[string]string); ok {
		*strMap = make(map[string]string)
		for _, key := range keys {
			if val, found := r.cache.Get(key); found {
				if strVal, ok := val.(string); ok {
					(*strMap)[key] = strVal
				}
			}
		}
		return nil
	}

	return &InvalidArgumentError{"dstMap must be a *map[string]string for this simple implementation"}
}

// Exists 批量检查键存在性
func (r *ristrettoStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	results := make(map[string]bool)
	for _, key := range keys {
		_, found := r.cache.Get(key)
		results[key] = found
	}
	return results, nil
}

// MSet 批量设置键值对，支持TTL
func (r *ristrettoStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	for key, value := range items {
		// For simplicity, we're assuming values are strings
		// In a real implementation, you'd need to handle serialization
		if strVal, ok := value.(string); ok {
			// Ristretto doesn't have a direct TTL per item, but we can use the cost parameter
			// For this simple implementation, we'll use a cost of 1 for all items
			// A more sophisticated implementation would handle TTL differently
			r.cache.Set(key, strVal, 1)
		} else {
			return &InvalidArgumentError{"values must be strings for this simple implementation"}
		}
	}
	return nil
}

// Del 删除指定键
func (r *ristrettoStore) Del(ctx context.Context, keys ...string) (int64, error) {
	var count int64
	for _, key := range keys {
		// Del always returns nil, so we can't use its return value to determine if a key was deleted.
		// We'll assume the key existed and was deleted for simplicity in this mock.
		r.cache.Del(key)
		count++
	}
	return count, nil
}

func TestRistrettoStore_Get(t *testing.T) {
	// Create a Ristretto cache
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1000,
		MaxCost:     100,
		BufferItems: 64,
	})
	assert.NoError(t, err)
	defer cache.Close()

	// Create a Ristretto store
	store := NewRistrettoStore(cache)

	// Test case 1: Key exists
	t.Run("KeyExists", func(t *testing.T) {
		// Set a value in Ristretto
		cache.Set("key1", "value1", 1)
		cache.Wait()

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

func TestRistrettoStore_MGet(t *testing.T) {
	// Create a Ristretto cache
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1000,
		MaxCost:     100,
		BufferItems: 64,
	})
	assert.NoError(t, err)
	defer cache.Close()

	// Create a Ristretto store
	store := NewRistrettoStore(cache)

	// Set some values in Ristretto
	cache.Set("key1", "value1", 1)
	cache.Set("key2", "value2", 1)
	cache.Wait()

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

func TestRistrettoStore_Exists(t *testing.T) {
	// Create a Ristretto cache
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1000,
		MaxCost:     100,
		BufferItems: 64,
	})
	assert.NoError(t, err)
	defer cache.Close()

	// Create a Ristretto store
	store := NewRistrettoStore(cache)

	// Set a value in Ristretto
	cache.Set("key1", "value1", 1)
	cache.Wait()

	// Test case: Check existence
	t.Run("CheckExistence", func(t *testing.T) {
		results, err := store.Exists(context.Background(), []string{"key1", "nonexistent"})
		assert.NoError(t, err)
		assert.True(t, results["key1"])
		assert.False(t, results["nonexistent"])
	})
}

func TestRistrettoStore_MSet(t *testing.T) {
	// Create a Ristretto cache
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1000,
		MaxCost:     100,
		BufferItems: 64,
	})
	assert.NoError(t, err)
	defer cache.Close()

	// Create a Ristretto store
	store := NewRistrettoStore(cache)

	// Test case: Set values
	t.Run("SetValues", func(t *testing.T) {
		items := map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		}
		err := store.MSet(context.Background(), items, 0)
		assert.NoError(t, err)

		// Wait for the items to be set
		time.Sleep(10 * time.Millisecond)

		// Verify the values were set
		val1, found := cache.Get("key1")
		assert.True(t, found)
		assert.Equal(t, "value1", val1)

		val2, found := cache.Get("key2")
		assert.True(t, found)
		assert.Equal(t, "value2", val2)
	})
}

func TestRistrettoStore_Del(t *testing.T) {
	// Create a Ristretto cache
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1000,
		MaxCost:     100,
		BufferItems: 64,
	})
	assert.NoError(t, err)
	defer cache.Close()

	// Create a Ristretto store
	store := NewRistrettoStore(cache)

	// Set some values in Ristretto
	cache.Set("key1", "value1", 1)
	cache.Set("key2", "value2", 1)
	cache.Set("key3", "value3", 1)
	cache.Wait()

	// Test case: Delete keys
	t.Run("DeleteKeys", func(t *testing.T) {
		count, err := store.Del(context.Background(), "key1", "key2", "nonexistent")
		assert.NoError(t, err)
		assert.Equal(t, int64(2), count)

		// Verify the keys were deleted
		_, found := cache.Get("key1")
		assert.False(t, found)

		_, found = cache.Get("key2")
		assert.False(t, found)

		// key3 should still exist
		val, found := cache.Get("key3")
		assert.True(t, found)
		assert.Equal(t, "value3", val)
	})
}