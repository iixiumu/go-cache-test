package ristretto

import (
	"context"
	"testing"
	"time"

	"go-cache/cacher/store"
)

// TestRistrettoStore runs the unified test suite for RistrettoStore
func TestRistrettoStore(t *testing.T) {
	// Create a RistrettoStore
	ristrettoStore, err := NewRistrettoStore()
	if err != nil {
		t.Fatalf("Failed to create RistrettoStore: %v", err)
	}

	// Create a StoreTester
	tester := &store.StoreTester{
		NewStore: func() store.Store {
			return ristrettoStore
		},
	}

	// Run all tests
	tester.RunAllTests(t)
}

// TestRistrettoStoreSpecific tests Ristretto-specific functionality
func TestRistrettoStoreSpecific(t *testing.T) {
	// Create a RistrettoStore
	store, err := NewRistrettoStore()
	if err != nil {
		t.Fatalf("Failed to create RistrettoStore: %v", err)
	}

	ctx := context.Background()

	// Test TTL functionality specifically
	key := "ttl_test_key"
	value := "ttl_test_value"
	items := map[string]interface{}{key: value}

	// Set with 100ms TTL
	err = store.MSet(ctx, items, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	// Verify value exists initially
	var result string
	found, err := store.Get(ctx, key, &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Fatalf("Key not found")
	}
	if result != value {
		t.Fatalf("Expected %v, got %v", value, result)
	}

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Verify value has expired
	found, err = store.Get(ctx, key, &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if found {
		t.Fatalf("Expected key to be expired")
	}
}