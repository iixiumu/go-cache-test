package ristretto

import (
	"context"
	"testing"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRistretto(t *testing.T) *RistrettoStore {
	config := &ristretto.Config{
		NumCounters: 1000,
		MaxCost:     1 << 20, // 1MB
		BufferItems: 64,
	}
	
	store, err := NewRistrettoStore(config)
	require.NoError(t, err)
	
	return store
}

func TestRistrettoStore_Get(t *testing.T) {
	store := setupTestRistretto(t)
	defer store.Close()

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

func TestRistrettoStore_MGet(t *testing.T) {
	store := setupTestRistretto(t)
	defer store.Close()

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

func TestRistrettoStore_Exists(t *testing.T) {
	store := setupTestRistretto(t)
	defer store.Close()

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

func TestRistrettoStore_MSet(t *testing.T) {
	store := setupTestRistretto(t)
	defer store.Close()

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

func TestRistrettoStore_MSetWithTTL(t *testing.T) {
	store := setupTestRistretto(t)
	defer store.Close()

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

func TestRistrettoStore_Del(t *testing.T) {
	store := setupTestRistretto(t)
	defer store.Close()

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

func TestRistrettoStore_ComplexTypes(t *testing.T) {
	store := setupTestRistretto(t)
	defer store.Close()

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

func TestNewDefaultRistrettoStore(t *testing.T) {
	store, err := NewDefaultRistrettoStore()
	assert.NoError(t, err)
	assert.NotNil(t, store)
	defer store.Close()
	
	ctx := context.Background()
	
	// Basic functionality test
	items := map[string]interface{}{
		"test": "value",
	}
	err = store.MSet(ctx, items, 0)
	assert.NoError(t, err)
	
	var result string
	found, err := store.Get(ctx, "test", &result)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "value", result)
}