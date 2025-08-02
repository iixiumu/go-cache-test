package cache

import (
	"context"
	"reflect"
	"testing"
	"time"
)

// TestStoreImplementation verifies the Store interface contract
func verifyStoreImplementation(t *testing.T, store Store) {
	ctx := context.Background()

	// Test MSet and Get
	items := map[string]interface{}{
		"test1": "value1",
		"test2": 123,
	}
	if err := store.MSet(ctx, items, 10*time.Second); err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	var val1 string
	found, err := store.Get(ctx, "test1", &val1)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found || val1 != "value1" {
		t.Errorf("Expected value1, got %v", val1)
	}

	// Test MGet
	var resultMap map[string]interface{}
	if err := store.MGet(ctx, []string{"test1", "test2"}, &resultMap); err != nil {
		t.Fatalf("MGet failed: %v", err)
	}
	if len(resultMap) != 2 {
		t.Errorf("Expected 2 results, got %d", len(resultMap))
	}

	// Test Exists
	existsMap, err := store.Exists(ctx, []string{"test1", "test3"})
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !existsMap["test1"] || existsMap["test3"] {
		t.Errorf("Unexpected existence results: %v", existsMap)
	}

	// Test Del
	deleted, err := store.Del(ctx, "test1", "test2")
	if err != nil {
		t.Fatalf("Del failed: %v", err)
	}
	if deleted != 2 {
		t.Errorf("Expected 2 deleted, got %d", deleted)
	}

	// Verify deletion
	var valAfter string
	found, _ = store.Get(ctx, "test1", &valAfter)
	if found {
		t.Error("Key test1 should have been deleted")
	}
}

func verifyStoreTypeHandling(t *testing.T, store Store) {
	ctx := context.Background()
	// Test type preservation
	if err := store.MSet(ctx, map[string]interface{}{
		"int":   42,
		"slice": []int{1, 2, 3},
	}, 0); err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	var intValue int
	if _, err := store.Get(ctx, "int", &intValue); err != nil || intValue != 42 {
		t.Error("Failed to retrieve int value correctly")
	}

	var sliceValue []int
	if _, err := store.Get(ctx, "slice", &sliceValue); err != nil || !reflect.DeepEqual(sliceValue, []int{1, 2, 3}) {
		t.Error("Failed to retrieve slice value correctly")
	}
}