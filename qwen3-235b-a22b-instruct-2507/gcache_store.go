package cache

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/bluele/gcache"
)

// GCacheStore implements the Store interface using gcache as the backend
type GCacheStore struct {
	cache gcache.Cache
}

// NewGCacheStore creates a new GCacheStore with the given gcache cache
func NewGCacheStore(cache gcache.Cache) *GCacheStore {
	return &GCacheStore{cache: cache}
}

// Get implements the Get method of Store interface
func (g *GCacheStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	// Get value from cache
	value, err := g.cache.Get(key)
	if err != nil {
		if err == gcache.KeyNotFoundError {
			return false, nil
		}
		return false, err
	}
	
	// Value is stored as JSON bytes, unmarshal to destination
	if err := json.Unmarshal(value.([]byte), dst); err != nil {
		return false, err
	}
	
	return true, nil
}

// MGet implements the MGet method of Store interface
func (g *GCacheStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
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
		value, err := g.cache.Get(key)
		if err != nil {
			if err == gcache.KeyNotFoundError {
				continue
			}
			return err
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
func (g *GCacheStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool)
	for _, key := range keys {
		_, err := g.cache.Get(key)
		if err != nil {
			if err == gcache.KeyNotFoundError {
				result[key] = false
			} else {
				return nil, err
			}
		} else {
			result[key] = true
		}
	}
	return result, nil
}

// MSet implements the MSet method of Store interface
func (g *GCacheStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	// Convert items to a format suitable for gcache
	for k, v := range items {
		// Marshal value to JSON
		val, err := json.Marshal(v)
		if err != nil {
			return err
		}
		
		// Set the item in cache
		if ttl > 0 {
			g.cache.SetWithExpire(k, val, ttl)
		} else {
			g.cache.Set(k, val)
		}
	}
	
	return nil
}

// Del implements the Del method of Store interface
func (g *GCacheStore) Del(ctx context.Context, keys ...string) (int64, error) {
	var deleted int64
	for _, key := range keys {
		if g.cache.Remove(key) {
			deleted++
		}
	}
	return deleted, nil
}

// TestGCacheStore tests the GCacheStore implementation
func TestGCacheStore(t *testing.T) {
	// Create a new gcache cache (using simple cache for testing)
	cache := gcache.New(1000).
		Simple().
		Build()
	
	store := NewGCacheStore(cache)
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