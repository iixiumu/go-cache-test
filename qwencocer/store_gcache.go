package cache

import (
	"context"
	"testing"
	"time"

	"github.com/bluele/gcache"
	"github.com/stretchr/testify/assert"
)

// gcacheStore is a gcache implementation of the Store interface
type gcacheStore struct {
	cache gcache.Cache
}

// NewGCacheStore creates a new gcache store
func NewGCacheStore(cache gcache.Cache) Store {
	return &gcacheStore{cache: cache}
}

// Get 从存储后端获取单个值
func (g *gcacheStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	val, err := g.cache.Get(key)
	if err == gcache.KeyNotFoundError {
		return false, nil
	}
	if err != nil {
		return false, err
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
func (g *gcacheStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	// For simplicity, we're assuming dstMap is a *map[string]string
	// In a real implementation, you'd need to handle different types with reflection
	if strMap, ok := dstMap.(*map[string]string); ok {
		*strMap = make(map[string]string)
		for _, key := range keys {
			val, err := g.cache.Get(key)
			if err == nil {
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
func (g *gcacheStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	results := make(map[string]bool)
	for _, key := range keys {
		_, err := g.cache.Get(key)
		results[key] = err == nil
	}
	return results, nil
}

// MSet 批量设置键值对，支持TTL
func (g *gcacheStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	for key, value := range items {
		// For simplicity, we're assuming values are strings
		// In a real implementation, you'd need to handle serialization
		if strVal, ok := value.(string); ok {
			if ttl > 0 {
				g.cache.SetWithExpire(key, strVal, ttl)
			} else {
				g.cache.Set(key, strVal)
			}
		} else {
			return &InvalidArgumentError{"values must be strings for this simple implementation"}
		}
	}
	return nil
}

// Del 删除指定键
func (g *gcacheStore) Del(ctx context.Context, keys ...string) (int64, error) {
	var count int64
	for _, key := range keys {
		if g.cache.Remove(key) {
			count++
		}
	}
	return count, nil
}

func TestGCacheStore_Get(t *testing.T) {
	// Create a gcache instance
	gc := gcache.New(100).Build()

	// Create a gcache store
	store := NewGCacheStore(gc)

	// Test case 1: Key exists
	t.Run("KeyExists", func(t *testing.T) {
		// Set a value in gcache
		gc.Set("key1", "value1")

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

func TestGCacheStore_MGet(t *testing.T) {
	// Create a gcache instance
	gc := gcache.New(100).Build()

	// Create a gcache store
	store := NewGCacheStore(gc)

	// Set some values in gcache
	gc.Set("key1", "value1")
	gc.Set("key2", "value2")

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

func TestGCacheStore_Exists(t *testing.T) {
	// Create a gcache instance
	gc := gcache.New(100).Build()

	// Create a gcache store
	store := NewGCacheStore(gc)

	// Set a value in gcache
	gc.Set("key1", "value1")

	// Test case: Check existence
	t.Run("CheckExistence", func(t *testing.T) {
		results, err := store.Exists(context.Background(), []string{"key1", "nonexistent"})
		assert.NoError(t, err)
		assert.True(t, results["key1"])
		assert.False(t, results["nonexistent"])
	})
}

func TestGCacheStore_MSet(t *testing.T) {
	// Create a gcache instance
	gc := gcache.New(100).Build()

	// Create a gcache store
	store := NewGCacheStore(gc)

	// Test case 1: Set values without TTL
	t.Run("SetValuesWithoutTTL", func(t *testing.T) {
		items := map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		}
		err := store.MSet(context.Background(), items, 0)
		assert.NoError(t, err)

		// Verify the values were set
		val1, err := gc.Get("key1")
		assert.NoError(t, err)
		assert.Equal(t, "value1", val1)

		val2, err := gc.Get("key2")
		assert.NoError(t, err)
		assert.Equal(t, "value2", val2)
	})

	// Test case 2: Set values with TTL
	// Note: gcache TTL testing is more complex and might require a more sophisticated approach
	// For simplicity, we're not implementing a TTL test here
}

func TestGCacheStore_Del(t *testing.T) {
	// Create a gcache instance
	gc := gcache.New(100).Build()

	// Create a gcache store
	store := NewGCacheStore(gc)

	// Set some values in gcache
	gc.Set("key1", "value1")
	gc.Set("key2", "value2")
	gc.Set("key3", "value3")

	// Test case: Delete keys
	t.Run("DeleteKeys", func(t *testing.T) {
		count, err := store.Del(context.Background(), "key1", "key2", "nonexistent")
		assert.NoError(t, err)
		assert.Equal(t, int64(2), count)

		// Verify the keys were deleted
		_, err = gc.Get("key1")
		assert.Equal(t, gcache.KeyNotFoundError, err)

		_, err = gc.Get("key2")
		assert.Equal(t, gcache.KeyNotFoundError, err)

		// key3 should still exist
		val, err := gc.Get("key3")
		assert.NoError(t, err)
		assert.Equal(t, "value3", val)
	})
}