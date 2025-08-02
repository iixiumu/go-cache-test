package deepseek

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMemoryStore_GetSet(t *testing.T) {
	store := NewMemoryStore()

	t.Run("set and get", func(t *testing.T) {
		err := store.MSet(context.Background(), map[string]interface{}{"key1": "value1"}, 0)
		assert.NoError(t, err)

		var dst string
		found, err := store.Get(context.Background(), "key1", &dst)
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "value1", dst)
	})

	t.Run("expiration", func(t *testing.T) {
		err := store.MSet(context.Background(), map[string]interface{}{"exp_key": "value"}, time.Millisecond*100)
		assert.NoError(t, err)

		time.Sleep(time.Millisecond * 150)

		var dst string
		found, err := store.Get(context.Background(), "exp_key", &dst)
		assert.NoError(t, err)
		assert.False(t, found)
	})

	t.Run("delete", func(t *testing.T) {
		err := store.MSet(context.Background(), map[string]interface{}{"del_key": "value"}, 0)
		assert.NoError(t, err)

		count, err := store.Del(context.Background(), "del_key")
		assert.NoError(t, err)
		assert.Equal(t, int64(1), count)

		var dst string
		found, err := store.Get(context.Background(), "del_key", &dst)
		assert.NoError(t, err)
		assert.False(t, found)
	})
}