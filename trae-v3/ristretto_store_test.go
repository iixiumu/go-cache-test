package deepseek

import (
	"context"
	"testing"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/stretchr/testify/assert"
)

func TestRistrettoStore(t *testing.T) {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,
		MaxCost:     1 << 30,
		BufferItems: 64,
	})
	if err != nil {
		t.Fatal(err)
	}

	store := NewRistrettoStore(cache)

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
}