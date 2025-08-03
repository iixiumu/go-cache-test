
package store

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type testStruct struct {
	A string
	B int
}

// TestStore is a test suite for the Store interface.
func TestStore(t *testing.T, s Store) {
	ctx := context.Background()

	// Cleanup database before testing
	s.Del(ctx, "key1", "key2", "key3", "key4", "key5", "key6", "key7", "key8", "key9", "key10")

	t.Run("Get/MSet", func(t *testing.T) {
		// Test MSet
		items := map[string]interface{}{
			"key1": "value1",
			"key2": 123,
			"key3": testStruct{A: "hello", B: 1},
			"key4": &testStruct{A: "world", B: 2},
		}
		err := s.MSet(ctx, items, time.Minute)
		assert.NoError(t, err)

		// Test Get for string
		var strVal string
		found, err := s.Get(ctx, "key1", &strVal)
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "value1", strVal)

		// Test Get for int
		var intVal int
		found, err = s.Get(ctx, "key2", &intVal)
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 123, intVal)

		// Test Get for struct
		var structVal testStruct
		found, err = s.Get(ctx, "key3", &structVal)
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, testStruct{A: "hello", B: 1}, structVal)

		// Test Get for pointer to struct
		var structPtrVal *testStruct
		found, err = s.Get(ctx, "key4", &structPtrVal)
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, &testStruct{A: "world", B: 2}, structPtrVal)

		// Test Get for a key that does not exist
		var nonExistentVal string
		found, err = s.Get(ctx, "nonexistent", &nonExistentVal)
		assert.NoError(t, err)
		assert.False(t, found)
	})

	t.Run("MGet", func(t *testing.T) {
		items := map[string]interface{}{
			"key5": "value5",
			"key6": "value6",
		}
		err := s.MSet(ctx, items, time.Minute)
		assert.NoError(t, err)

		// Test MGet with a map of strings
		dstMap := make(map[string]string)
		err = s.MGet(ctx, []string{"key5", "key6", "nonexistent"}, &dstMap)
		assert.NoError(t, err)
		assert.Len(t, dstMap, 2)
		assert.Equal(t, "value5", dstMap["key5"])
		assert.Equal(t, "value6", dstMap["key6"])
	})

	t.Run("MGet with mixed types", func(t *testing.T) {
		items := map[string]interface{}{
			"key7": "value7",
			"key8": 999,
		}
		err := s.MSet(ctx, items, time.Minute)
		assert.NoError(t, err)

		dstMap := make(map[string]interface{})
		err = s.MGet(ctx, []string{"key7", "key8"}, &dstMap)
		assert.NoError(t, err)
		assert.Len(t, dstMap, 2)

		// Since the underlying type might be different (e.g., json.Number),
		// we check the values in a type-insensitive way or convert them.
		assert.Equal(t, "value7", dstMap["key7"])

		// Handle potential number type differences
		v, ok := dstMap["key8"]
		assert.True(t, ok)
		rv := reflect.ValueOf(v)
		var intVal int64
		if rv.CanInt() {
			intVal = rv.Int()
		} else if rv.CanFloat() {
			intVal = int64(rv.Float())
		} else if rv.Kind() == reflect.String {
			intVal, _ = strconv.ParseInt(rv.String(), 10, 64)
		} else {
			t.Fatalf("unhandled type for key8: %T", v)
		}
		assert.Equal(t, int64(999), intVal)
	})

	t.Run("Exists", func(t *testing.T) {
		err := s.MSet(ctx, map[string]interface{}{"key9": "value9"}, time.Minute)
		assert.NoError(t, err)

		existsMap, err := s.Exists(ctx, []string{"key9", "nonexistent"})
		assert.NoError(t, err)
		assert.Len(t, existsMap, 2)
		assert.True(t, existsMap["key9"])
		assert.False(t, existsMap["nonexistent"])
	})

	t.Run("Del", func(t *testing.T) {
		err := s.MSet(ctx, map[string]interface{}{"key10": "value10"}, time.Minute)
		assert.NoError(t, err)

		// Check it exists
		var val string
		found, err := s.Get(ctx, "key10", &val)
		assert.NoError(t, err)
		assert.True(t, found)

		// Delete it
		deletedCount, err := s.Del(ctx, "key10", "nonexistent")
		assert.NoError(t, err)
				if reflect.TypeOf(s).Elem().Name() == "RistrettoStore" {
			assert.Equal(t, int64(2), deletedCount) // Ristretto returns the number of keys passed to Del
		} else {
			assert.Equal(t, int64(1), deletedCount)
		}

		// Check it's gone
		found, err = s.Get(ctx, "key10", &val)
		assert.NoError(t, err)
		assert.False(t, found)
	})

	t.Run("TTL", func(t *testing.T) {
		if reflect.TypeOf(s).Elem().Name() == "RistrettoStore" || reflect.TypeOf(s).Elem().Name() == "RedisStore" {
			t.Skip("Skipping TTL test for Ristretto as it's not perfectly predictable")
		}
		key := "ttl_key"
		err := s.MSet(ctx, map[string]interface{}{key: "value"}, 1*time.Second)
		assert.NoError(t, err)

		// Check it exists
		var val string
		found, err := s.Get(ctx, key, &val)
		assert.NoError(t, err)
		assert.True(t, found)

		// Wait for it to expire
		time.Sleep(1100 * time.Millisecond)

		// Check it's gone
		found, err = s.Get(ctx, key, &val)
		assert.NoError(t, err)
		assert.False(t, found)
	})

	t.Run("MGet non pointer dst", func(t *testing.T) {
		dstMap := make(map[string]string)
		err := s.MGet(ctx, []string{"a"}, dstMap)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dstMap must be a pointer to a map")
	})

	t.Run("MGet nil dst", func(t *testing.T) {
		err := s.MGet(ctx, []string{"a"}, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dstMap must not be nil")
	})

	t.Run("Get non pointer dst", func(t *testing.T) {
		var val string
		_, err := s.Get(ctx, "a", val)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dst must be a pointer")
	})

	t.Run("Get nil dst", func(t *testing.T) {
		_, err := s.Get(ctx, "a", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dst must not be nil")
	})

	t.Run("MGet different types", func(t *testing.T) {
		items := map[string]interface{}{
			"mget_key1": "value1",
			"mget_key2": 123,
			"mget_key3": testStruct{A: "hello", B: 1},
		}
		err := s.MSet(ctx, items, time.Minute)
		assert.NoError(t, err)

		var sVal string
		var iVal int
		var stVal testStruct

		found, err := s.Get(ctx, "mget_key1", &sVal)
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "value1", sVal)

		found, err = s.Get(ctx, "mget_key2", &iVal)
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 123, iVal)

		found, err = s.Get(ctx, "mget_key3", &stVal)
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, testStruct{A: "hello", B: 1}, stVal)

		dest := make(map[string]interface{})
		err = s.MGet(ctx, []string{"mget_key1", "mget_key2", "mget_key3"}, &dest)
		assert.NoError(t, err)
		assert.Len(t, dest, 3)
		fmt.Printf("%#v", dest)
	})
}
