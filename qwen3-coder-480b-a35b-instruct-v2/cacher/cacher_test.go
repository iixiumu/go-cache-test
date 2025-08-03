package cacher

import (
	"context"
	"testing"

	"go-cache/cacher/store/ristretto"
)

// TestCacherGet tests the Get method of the Cacher interface
func TestCacherGet(t *testing.T) {
	// Create a Ristretto store for testing
	store, err := ristretto.NewRistrettoStore()
	if err != nil {
		t.Fatalf("Failed to create Ristretto store: %v", err)
	}

	// Create a Cacher instance
	cacher := NewCacher(store)

	ctx := context.Background()

	// Test Get with fallback function
	key := "test_key"
	expectedValue := "test_value"

	// Define a fallback function
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		return expectedValue, true, nil
	}

	// Get value using cacher
	var result string
	found, err := cacher.Get(ctx, key, &result, fallback, nil)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Fatalf("Expected value to be found")
	}
	if result != expectedValue {
		t.Fatalf("Expected %v, got %v", expectedValue, result)
	}

	// Get value again - should come from cache this time
	var result2 string
	found, err = cacher.Get(ctx, key, &result2, nil, nil) // No fallback
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Fatalf("Expected value to be found in cache")
	}
	if result2 != expectedValue {
		t.Fatalf("Expected %v, got %v", expectedValue, result2)
	}
}

// TestCacherGetWithoutFallback tests Get when value is not in cache and no fallback is provided
func TestCacherGetWithoutFallback(t *testing.T) {
	// Create a Ristretto store for testing
	store, err := ristretto.NewRistrettoStore()
	if err != nil {
		t.Fatalf("Failed to create Ristretto store: %v", err)
	}

	// Create a Cacher instance
	cacher := NewCacher(store)

	ctx := context.Background()

	// Try to get a non-existent key without fallback
	var result string
	found, err := cacher.Get(ctx, "non_existent_key", &result, nil, nil)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if found {
		t.Fatalf("Expected value not to be found")
	}
}

// TestCacherMGet tests the MGet method of the Cacher interface
func TestCacherMGet(t *testing.T) {
	// Create a Ristretto store for testing
	store, err := ristretto.NewRistrettoStore()
	if err != nil {
		t.Fatalf("Failed to create Ristretto store: %v", err)
	}

	// Create a Cacher instance
	cacher := NewCacher(store)

	ctx := context.Background()

	// Set some values directly in the store
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	err = store.MSet(ctx, items, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// Define a fallback function for missing keys
	fallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		result := make(map[string]interface{})
		for _, key := range keys {
			if key == "key3" {
				result[key] = "value3"
			}
		}
		return result, nil
	}

	// Get multiple values
	keys := []string{"key1", "key2", "key3"}
	resultMap := make(map[string]string)
	err = cacher.MGet(ctx, keys, &resultMap, fallback, nil)
	if err != nil {
		t.Fatalf("MGet failed: %v", err)
	}

	// Check results
	if len(resultMap) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(resultMap))
	}
	if resultMap["key1"] != "value1" {
		t.Fatalf("Expected value1, got %v", resultMap["key1"])
	}
	if resultMap["key2"] != "value2" {
		t.Fatalf("Expected value2, got %v", resultMap["key2"])
	}
	if resultMap["key3"] != "value3" {
		t.Fatalf("Expected value3, got %v", resultMap["key3"])
	}
}

// TestCacherMDelete tests the MDelete method of the Cacher interface
func TestCacherMDelete(t *testing.T) {
	// Create a Ristretto store for testing
	store, err := ristretto.NewRistrettoStore()
	if err != nil {
		t.Fatalf("Failed to create Ristretto store: %v", err)
	}

	// Create a Cacher instance
	cacher := NewCacher(store)

	ctx := context.Background()

	// Set some values
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	err = store.MSet(ctx, items, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// Delete some keys
	deleted, err := cacher.MDelete(ctx, []string{"key1", "key2", "non_existent_key"})
	if err != nil {
		t.Fatalf("MDelete failed: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("Expected 2 deletions, got %d", deleted)
	}

	// Verify keys are deleted
	var result string
	found, err := cacher.Get(ctx, "key1", &result, nil, nil)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if found {
		t.Fatalf("Expected key1 to be deleted")
	}
}

// TestCacherMRefresh tests the MRefresh method of the Cacher interface
func TestCacherMRefresh(t *testing.T) {
	// Create a Ristretto store for testing
	store, err := ristretto.NewRistrettoStore()
	if err != nil {
		t.Fatalf("Failed to create Ristretto store: %v", err)
	}

	// Create a Cacher instance
	cacher := NewCacher(store)

	ctx := context.Background()

	// Set initial values
	items := map[string]interface{}{
		"key1": "old_value1",
		"key2": "old_value2",
	}
	err = store.MSet(ctx, items, 0)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// Define a fallback function to refresh values
	fallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		result := make(map[string]interface{})
		for _, key := range keys {
			result[key] = "new_" + key
		}
		return result, nil
	}

	// Refresh values
	resultMap := make(map[string]string)
	err = cacher.MRefresh(ctx, []string{"key1", "key2"}, &resultMap, fallback, nil)
	if err != nil {
		t.Fatalf("MRefresh failed: %v", err)
	}

	// Check that values were refreshed
	if len(resultMap) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(resultMap))
	}
	if resultMap["key1"] != "new_key1" {
		t.Fatalf("Expected new_key1, got %v", resultMap["key1"])
	}
	if resultMap["key2"] != "new_key2" {
		t.Fatalf("Expected new_key2, got %v", resultMap["key2"])
	}

	// Verify values were updated in cache
	var result string
	found, err := cacher.Get(ctx, "key1", &result, nil, nil)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Fatalf("Expected key1 to be found")
	}
	if result != "new_key1" {
		t.Fatalf("Expected new_key1, got %v", result)
	}
}