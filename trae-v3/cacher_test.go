package deepseek

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCacher_Get(t *testing.T) {
	store := NewMemoryStore()
	cacher := NewCacher(store)

	t.Run("cache hit", func(t *testing.T) {
		// 先设置缓存
		err := store.MSet(context.Background(), map[string]interface{}{"key1": "value1"}, 0)
		assert.NoError(t, err)

		var dst string
		found, err := cacher.Get(context.Background(), "key1", &dst, nil, nil)
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "value1", dst)
	})

	t.Run("cache miss with fallback", func(t *testing.T) {
		var dst string
		fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
			return "fallback_value", true, nil
		}

		found, err := cacher.Get(context.Background(), "key2", &dst, fallback, &CacheOptions{TTL: time.Minute})
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "fallback_value", dst)

		// 验证缓存是否设置
		found, err = cacher.Get(context.Background(), "key2", &dst, nil, nil)
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, "fallback_value", dst)
	})
}