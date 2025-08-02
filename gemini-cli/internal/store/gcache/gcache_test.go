package gcache

import (
    "context"
    "testing"
    "time"

    "github.com/bluele/gcache"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestGcacheStore(t *testing.T) {
    gc := gcache.New(20).LRU().Build()
    store := NewGcacheStore(gc)
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

        dstMap := make(map[string]interface{}) // Use interface{} to handle different types
        err = store.MGet(ctx, []string{"key1", "key2", "key3"}, &dstMap)
        assert.NoError(t, err)

        expected := map[string]interface{}{"key1": "value1", "key2": float64(123)}
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
        assert.Equal(t, int64(1), deleted)

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
