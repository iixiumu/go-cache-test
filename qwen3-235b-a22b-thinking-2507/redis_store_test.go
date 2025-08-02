package cache

import (
	"context"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisStore(t *testing.T) {
	// Start a miniredis server
	r, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer r.Close()

	// Create redis client
	client := redis.NewClient(&redis.Options{
		Addr: r.Addr(),
	})
	defer client.Close()

	// Create store
	store := NewRedisStore(client)

	// Run generic store tests
	verifyStoreImplementation(t, store)
	verifyStoreTypeHandling(t, store)

	// Test TTL behavior
	ctx := context.Background()
	if err := store.MSet(ctx, map[string]interface{}{
		"ttl_key": "value",
	}, 100*time.Millisecond); err != nil {
		t.Fatalf("MSet failed: %v", err)
	}

	var val string
	found, err := store.Get(ctx, "ttl_key", &val)
	if err != nil || !found {
		t.Error("Key should exist initially")
	}

	// Wait for key to expire
	time.Sleep(150 * time.Millisecond)

	found, _ = store.Get(ctx, "ttl_key", &val)
	if found {
		t.Error("Key should have expired")
	}
}