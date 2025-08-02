package cache

import (
	"context"
	"reflect"
	"testing"
	"time"
)

// TestStoreInterface tests the Store interface methods
func TestStoreInterface(t *testing.T) {
	// This is a test interface that any Store implementation should pass
	testStoreInterface := func(t *testing.T, store Store) {
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
		err = store.MSet(ctx, items, time.Minute)
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
	
	// We'll test this with actual implementations later
	// For now, we just define the test structure
	t.Run("MockStore", func(t *testing.T) {
		store := NewMockStore()
		testStoreInterface(t, store)
	})
}

// BenchmarkStoreGet benchmarks the Get method
func BenchmarkStoreGet(b *testing.B) {
	store := NewMockStore()
	ctx := context.Background()
	
	// Set up data
	store.MSet(ctx, map[string]interface{}{
		"benchmark_key": "benchmark_value",
	}, time.Minute)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result string
		store.Get(ctx, "benchmark_key", &result)
	}
}

// BenchmarkStoreMGet benchmarks the MGet method
func BenchmarkStoreMGet(b *testing.B) {
	store := NewMockStore()
	ctx := context.Background()
	
	// Set up data
	keys := make([]string, 100)
	items := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		key := "key_" + string(rune(i))
		keys[i] = key
		items[key] = "value_" + string(rune(i))
	}
	store.MSet(ctx, items, time.Minute)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var resultMap map[string]string
		store.MGet(ctx, keys, &resultMap)
	}
}

// BenchmarkStoreMSet benchmarks the MSet method
func BenchmarkStoreMSet(b *testing.B) {
	store := NewMockStore()
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		items := map[string]interface{}{
			"key_" + string(rune(i)): "value_" + string(rune(i)),
		}
		store.MSet(ctx, items, time.Minute)
	}
}