package redis

import (
	"context"
	"testing"

	store "go-cache/cacher/store"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// redisStoreTester Redis Store测试器
type redisStoreTester struct {
	mr *miniredis.Miniredis
}

func (r *redisStoreTester) NewStore() (store.Store, error) {
	return NewRedisStore(redis.NewClient(&redis.Options{
		Addr: r.mr.Addr(),
	})), nil
}

func (r *redisStoreTester) Name() string {
	return "Redis"
}

func (r *redisStoreTester) SetupTest(t *testing.T) {
	var err error
	r.mr, err = miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
}

func (r *redisStoreTester) TeardownTest(t *testing.T) {
	if r.mr != nil {
		r.mr.Close()
	}
}

func TestRedisStore(t *testing.T) {
	tester := &redisStoreTester{}
	store.RunStoreTests(t, tester)
}

func TestRedisStoreSpecific(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	redisStore := NewRedisStore(redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	}))

	ctx := context.Background()

	// Test various data types
	t.Run("TestString", func(t *testing.T) {
		key := "string_key"
		value := "test_string"

		err := redisStore.MSet(ctx, map[string]interface{}{key: value}, 0)
		if err != nil {
			t.Fatalf("MSet failed: %v", err)
		}

		var result string
		found, err := redisStore.Get(ctx, key, &result)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if !found {
			t.Error("Key should be found")
		}

		if result != value {
			t.Errorf("Expected %v, got %v", value, result)
		}
	})

	t.Run("TestInt", func(t *testing.T) {
		key := "int_key"
		value := 42

		err := redisStore.MSet(ctx, map[string]interface{}{key: value}, 0)
		if err != nil {
			t.Fatalf("MSet failed: %v", err)
		}

		var result int
		found, err := redisStore.Get(ctx, key, &result)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if !found {
			t.Error("Key should be found")
		}

		if result != value {
			t.Errorf("Expected %v, got %v", value, result)
		}
	})

	t.Run("TestStruct", func(t *testing.T) {
		type TestStruct struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}

		key := "struct_key"
		value := TestStruct{Name: "test", Value: 123}

		err := redisStore.MSet(ctx, map[string]interface{}{key: value}, 0)
		if err != nil {
			t.Fatalf("MSet failed: %v", err)
		}

		var result TestStruct
		found, err := redisStore.Get(ctx, key, &result)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if !found {
			t.Error("Key should be found")
		}

		if result.Name != value.Name || result.Value != value.Value {
			t.Errorf("Expected %+v, got %+v", value, result)
		}
	})
}
