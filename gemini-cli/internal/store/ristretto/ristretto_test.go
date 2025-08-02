package ristretto

import (
    "context"
    "encoding/json"
    "testing"
    "time"

    "github.com/dgraph-io/ristretto"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestRistrettoStore(t *testing.T) {
    cache, err := ristretto.NewCache(&ristretto.Config{
        NumCounters: 1e7,     // number of keys to track frequency of.
        MaxCost:     1 << 30, // maximum cost of cache.
        BufferItems: 64,      // number of keys per Get buffer.
    })
    require.NoError(t, err)

    store := NewRistrettoStore(cache)
    ctx := context.Background()

    t.Run("Get/MSet", func(t *testing.T) {
        err := store.MSet(ctx, map[string]interface{}{"key1": "value1"}, 0)
        require.NoError(t, err)

        var dst string
        found, err := store.Get(ctx, "key1", &dst)
        assert.NoError(t, err)
        assert.True(t, found)
        assert.Equal(t, "value1", dst)

        found, err = store.Get(ctx, "key_not_exist", &dst)
        assert.NoError(t, err)
        assert.False(t, found)
    })

    t.Run("MGet", func(t *testing.T) {
        err := store.MSet(ctx, map[string]interface{}{"key1": "value1", "key2": 123}, 0)
        require.NoError(t, err)

        dstMap := make(map[string]interface{})
        err = store.MGet(ctx, []string{"key1", "key2", "key3"}, &dstMap)
        assert.NoError(t, err)

        num, err := dstMap["key2"].(json.Number).Int64()
        assert.NoError(t, err)
        dstMap["key2"] = int(num)

        expected := map[string]interface{}{"key1": "value1", "key2": 123}
        assert.Equal(t, expected, dstMap)
    })

    t.Run("Exists", func(t *testing.T) {
        err := store.MSet(ctx, map[string]interface{}{"key1": "value1"}, 0)
        require.NoError(t, err)

        exists, err := store.Exists(ctx, []string{"key1", "key_not_exist"})
        assert.NoError(t, err)
        assert.Equal(t, map[string]bool{"key1": true, "key_not_exist": false}, exists)
    })

    t.Run("Del", func(t *testing.T) {
        err := store.MSet(ctx, map[string]interface{}{"key1": "value1", "key2": "value2"}, 0)
        require.NoError(t, err)

        deleted, err := store.Del(ctx, "key1", "key3")
        assert.NoError(t, err)
        assert.Equal(t, int64(2), deleted) // ristretto Del is not atomic, so it returns the number of keys passed

        var dst string
        found, err := store.Get(ctx, "key1", &dst)
        assert.NoError(t, err)
        assert.False(t, found)
    })

    t.Run("TTL", func(t *testing.T) {
        err := store.MSet(ctx, map[string]interface{}{"key_ttl": "value_ttl"}, 1*time.Millisecond)
        require.NoError(t, err)

        time.Sleep(10 * time.Millisecond)

        var dst string
        found, err := store.Get(ctx, "key_ttl", &dst)
        assert.NoError(t, err)
        assert.False(t, found)
    })
}
