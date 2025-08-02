package gcache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestGCache(t *testing.T) *GCacheStore {
	return NewLRUGCacheStore(100)
}

func TestGCacheStore_Get(t *testing.T) {
	store := setupTestGCache(t)

	ctx := context.Background()

	// Test getting non-existent key
	var result string
	found, err := store.Get(ctx, "nonexistent", &result)
	assert.NoError(t, err)
	assert.False(t, found)

	// Test setting and getting a key
	items := map[string]interface{}{
		"test_key": "test_value",
	}
	err = store.MSet(ctx, items, 0)
	require.NoError(t, err)

	found, err = store.Get(ctx, "test_key", &result)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "test_value", result)
}

func TestGCacheStore_MGet(t *testing.T) {
	store := setupTestGCache(t)

	ctx := context.Background()

	// Set test data
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	err := store.MSet(ctx, items, 0)
	require.NoError(t, err)

	// Test MGet
	keys := []string{"key1", "key2", "key4"} // key4 doesn't exist
	resultMap := make(map[string]string)
	err = store.MGet(ctx, keys, &resultMap)
	assert.NoError(t, err)

	assert.Equal(t, "value1", resultMap["key1"])
	assert.Equal(t, "value2", resultMap["key2"])
	_, exists := resultMap["key4"]
	assert.False(t, exists)
}

func TestGCacheStore_Exists(t *testing.T) {
	store := setupTestGCache(t)

	ctx := context.Background()

	// Set test data
	items := map[string]interface{}{
		"existing_key": "value",
	}
	err := store.MSet(ctx, items, 0)
	require.NoError(t, err)

	// Test Exists
	keys := []string{"existing_key", "nonexistent_key"}
	result, err := store.Exists(ctx, keys)
	assert.NoError(t, err)

	assert.True(t, result["existing_key"])
	assert.False(t, result["nonexistent_key"])
}

func TestGCacheStore_MSet(t *testing.T) {
	store := setupTestGCache(t)

	ctx := context.Background()

	// Test MSet without TTL
	items := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}
	err := store.MSet(ctx, items, 0)
	assert.NoError(t, err)

	// Verify the values
	var str string
	var num int
	var boolean bool

	found, err := store.Get(ctx, "key1", &str)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "value1", str)

	found, err = store.Get(ctx, "key2", &num)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, 123, num)

	found, err = store.Get(ctx, "key3", &boolean)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.True(t, boolean)
}

func TestGCacheStore_MSetWithTTL(t *testing.T) {
	store := setupTestGCache(t)

	ctx := context.Background()

	// Test MSet with TTL
	items := map[string]interface{}{
		"temp_key": "temp_value",
	}
	err := store.MSet(ctx, items, time.Millisecond*100)
	assert.NoError(t, err)

	// Verify the value exists initially
	var result string
	found, err := store.Get(ctx, "temp_key", &result)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "temp_value", result)

	// Wait for expiration and verify it's gone
	time.Sleep(time.Millisecond * 150)
	found, err = store.Get(ctx, "temp_key", &result)
	assert.NoError(t, err)
	assert.False(t, found)
}

func TestGCacheStore_Del(t *testing.T) {
	store := setupTestGCache(t)

	ctx := context.Background()

	// Set test data
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	err := store.MSet(ctx, items, 0)
	require.NoError(t, err)

	// Delete some keys
	deleted, err := store.Del(ctx, "key1", "key2", "nonexistent")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), deleted) // Only key1 and key2 existed

	// Verify deletion
	var result string
	found, err := store.Get(ctx, "key1", &result)
	assert.NoError(t, err)
	assert.False(t, found)

	found, err = store.Get(ctx, "key3", &result)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "value3", result)
}

func TestGCacheStore_ComplexTypes(t *testing.T) {
	store := setupTestGCache(t)

	ctx := context.Background()

	type TestStruct struct {
		Name  string `json:"name"`
		Age   int    `json:"age"`
		Email string `json:"email"`
	}

	original := TestStruct{
		Name:  "John Doe",
		Age:   30,
		Email: "john@example.com",
	}

	// Store complex type
	items := map[string]interface{}{
		"user": original,
	}
	err := store.MSet(ctx, items, 0)
	assert.NoError(t, err)

	// Retrieve complex type
	var retrieved TestStruct
	found, err := store.Get(ctx, "user", &retrieved)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, original, retrieved)
}

func TestGCacheStore_DifferentCacheTypes(t *testing.T) {
	testCases := []struct {
		name      string
		cacheType string
	}{
		{"LRU", "lru"},
		{"LFU", "lfu"},
		{"ARC", "arc"},
		{"Default", "invalid"}, // Should default to LRU
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := NewGCacheStore(10, tc.cacheType)
			assert.NotNil(t, store)

			ctx := context.Background()
			items := map[string]interface{}{
				"test": "value",
			}
			err := store.MSet(ctx, items, 0)
			assert.NoError(t, err)

			var result string
			found, err := store.Get(ctx, "test", &result)
			assert.NoError(t, err)
			assert.True(t, found)
			assert.Equal(t, "value", result)
		})
	}
}

func TestGCacheStore_Len(t *testing.T) {
	store := setupTestGCache(t)

	ctx := context.Background()

	// Initially empty
	assert.Equal(t, 0, store.Len())

	// Add some items
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	err := store.MSet(ctx, items, 0)
	require.NoError(t, err)

	assert.Equal(t, 2, store.Len())

	// Delete one item
	deleted, err := store.Del(ctx, "key1")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), deleted)
	assert.Equal(t, 1, store.Len())
}

func TestGCacheStore_Purge(t *testing.T) {
	store := setupTestGCache(t)

	ctx := context.Background()

	// Add some items
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	err := store.MSet(ctx, items, 0)
	require.NoError(t, err)

	assert.Equal(t, 2, store.Len())

	// Purge all items
	store.Purge()
	assert.Equal(t, 0, store.Len())

	// Verify items are gone
	var result string
	found, err := store.Get(ctx, "key1", &result)
	assert.NoError(t, err)
	assert.False(t, found)
}