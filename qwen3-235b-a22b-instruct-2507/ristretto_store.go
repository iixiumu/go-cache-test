package cache

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/dgraph-io/ristretto"
)

// RistrettoStore implements the Store interface using ristretto as the backend
type RistrettoStore struct {
	cache *ristretto.Cache
}

// NewRistrettoStore creates a new RistrettoStore with the given ristretto cache
func NewRistrettoStore(cache *ristretto.Cache) *RistrettoStore {
	return &RistrettoStore{cache: cache}
}

// Get implements the Get method of Store interface
func (r *RistrettoStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	// Get value from cache
	value, found := r.cache.Get(key)
	if !found {
		return false, nil
	}
	
	// Value is stored as JSON bytes, unmarshal to destination
	if err := json.Unmarshal(value.([]byte), dst); err != nil {
		return false, err
	}
	
	return true, nil
}

// MGet implements the MGet method of Store interface
func (r *RistrettoStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
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
	
	// Process each key
	for _, key := range keys {
		// Get value from cache
		value, found := r.cache.Get(key)
		if !found {
			continue
		}
		
		// Create a temporary variable to unmarshal into
		tempVal := reflect.New(mapVal.Type().Elem())
		
		// Unmarshal JSON to temporary variable
		if err := json.Unmarshal(value.([]byte), tempVal.Interface()); err != nil {
			return err
		}
		
		// Set the value in the map
		mapVal.SetMapIndex(reflect.ValueOf(key), tempVal.Elem())
	}
	
	return nil
}

// Exists implements the Exists method of Store interface
func (r *RistrettoStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool)
	for _, key := range keys {
		_, found := r.cache.Get(key)
		result[key] = found
	}
	return result, nil
}

// MSet implements the MSet method of Store interface
func (r *RistrettoStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	// Convert items to a format suitable for ristretto
	for k, v := range items {
		// Marshal value to JSON
		val, err := json.Marshal(v)
		if err != nil {
			return err
		}
		
		// In ristretto, cost is used for eviction policy
		// We'll use the size of the serialized value as the cost
		cost := int64(len(val))
		
		// Set the item in cache
		// ttl is converted to milliseconds for ristretto
		var expireMS int64
		if ttl > 0 {
			expireMS = ttl.Milliseconds()
		}
		
		r.cache.SetWithTTL(k, val, cost, time.Duration(expireMS)*time.Millisecond)
	}
	
	return nil
}

// Del implements the Del method of Store interface
func (r *RistrettoStore) Del(ctx context.Context, keys ...string) (int64, error) {
	var deleted int64
	for _, key := range keys {
		if r.cache.Del(key) {
			deleted++
		}
	}
	return deleted, nil
}

// TestRistrettoStore tests the RistrettoStore implementation
func TestRistrettoStore(t *testing.T) {
	// Create a new ristretto cache
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // number of keys to track frequency of (10M).
		MaxCost:     1 << 30, // maximum cost of cache (1GB).
		BufferItems: 64,      // number of keys per Get buffer.
	})
	if err != nil {
		t.Fatalf("Failed to create ristretto cache: %v", err)
	}
	defer cache.Close()
	
	store := NewRistrettoStore(cache)
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