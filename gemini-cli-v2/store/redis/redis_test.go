package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"go-cache/store"
)

func newTestRedisStore(t *testing.T) (store.Store, *miniredis.Miniredis) {
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("An error '%s' was not expected when opening a stub redis connection", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	return NewRedisStore(client), s
}

func TestRedisStore_Get(t *testing.T) {
	store, s := newTestRedisStore(t)
	defer s.Close()

	ctx := context.Background()
	key := "test_key"
	value := "test_value"

	// Test when key exists
	s.Set(key, `{"value":"test_value"}`)
	var dst string
	found, err := store.Get(ctx, key, &dst)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !found {
		t.Errorf("Expected to find key '%s'", key)
	}
	if dst != value {
		t.Errorf("Expected value '%s', got '%s'", value, dst)
	}

	// Test when key does not exist
	s.FlushAll()
	var dst2 string
	found, err = store.Get(ctx, "non_existent_key", &dst2)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if found {
		t.Errorf("Expected not to find key 'non_existent_key'")
	}
}

func TestRedisStore_MGet(t *testing.T) {
	store, s := newTestRedisStore(t)
	defer s.Close()

	ctx := context.Background()
	keys := []string{"key1", "key2", "key3"}
	expected := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	s.Set("key1", `{"value":"value1"}`)
	s.Set("key2", `{"value":"value2"}`)

	dstMap := make(map[string]string)
	err := store.MGet(ctx, keys, &dstMap)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(dstMap) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(dstMap))
	}

	for k, v := range expected {
		if dstMap[k] != v {
			t.Errorf("Expected value '%s' for key '%s', got '%s'", v, k, dstMap[k])
		}
	}
}

func TestRedisStore_Exists(t *testing.T) {
	store, s := newTestRedisStore(t)
	defer s.Close()

	ctx := context.Background()
	keys := []string{"key1", "key2", "key3"}

	s.Set("key1", "value1")
	s.Set("key3", "value3")

	exists, err := store.Exists(ctx, keys)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !exists["key1"] {
		t.Errorf("Expected key1 to exist")
	}
	if exists["key2"] {
		t.Errorf("Expected key2 not to exist")
	}
	if !exists["key3"] {
		t.Errorf("Expected key3 to exist")
	}
}

func TestRedisStore_MSet(t *testing.T) {
	store, s := newTestRedisStore(t)
	defer s.Close()

	ctx := context.Background()
	items := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	}
	ttl := 5 * time.Minute

	err := store.MSet(ctx, items, ttl)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	val1, _ := s.Get("key1")
	if val1 != `{"value":"value1"}` {
		t.Errorf("Expected value for key1 to be 'value1', got '%s'", val1)
	}

	val2, _ := s.Get("key2")
	if val2 != `{"value":123}` {
		t.Errorf("Expected value for key2 to be '123', got '%s'", val2)
	}

	s.FastForward(6 * time.Minute)
	if s.Exists("key1") {
		t.Errorf("Expected key1 to have expired")
	}
}

func TestRedisStore_Del(t *testing.T) {
	store, s := newTestRedisStore(t)
	defer s.Close()

	ctx := context.Background()
	keys := []string{"key1", "key2", "key3"}

	s.Set("key1", "value1")
	s.Set("key2", "value2")

	deleted, err := store.Del(ctx, keys...)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if deleted != 2 {
		t.Errorf("Expected to delete 2 keys, got %d", deleted)
	}

	if s.Exists("key1") {
		t.Errorf("Expected key1 to be deleted")
	}
	if s.Exists("key2") {
		t.Errorf("Expected key2 to be deleted")
	}
}