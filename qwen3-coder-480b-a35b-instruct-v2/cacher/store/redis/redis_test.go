package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"go-cache/cacher/store"
)

// TestRedisStore runs the unified test suite for RedisStore
func TestRedisStore(t *testing.T) {
	// Create a miniredis server
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer s.Close()

	// Create a Redis client
	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	// Create a StoreTester
	tester := &store.StoreTester{
		NewStore: func() store.Store {
			return NewRedisStore(client)
		},
	}

	// Run all tests
	tester.RunAllTests(t)
}

// TestRedisStoreTTLWithFastForward tests Redis TTL functionality with proper miniredis time advancement
func TestRedisStoreTTLWithFastForward(t *testing.T) {
	// Create a miniredis server
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer s.Close()

	// Create a Redis client
	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	store := NewRedisStore(client)
	ctx := context.Background()

	// Test TTL functionality specifically
	key := "ttl_test_key"
	value := "ttl_test_value"
	items := map[string]interface{}{key: value}

	// Set with 1 second TTL
	err = store.MSet(ctx, items, 1*time.Second)
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

	// Fast forward time by 1.5 seconds to simulate expiration
	// This is the correct way to test TTL in miniredis
	s.FastForward(1500 * time.Millisecond)

	// Verify value has expired
	found, err = store.Get(ctx, key, &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if found {
		t.Fatalf("Expected key to be expired")
	}
}