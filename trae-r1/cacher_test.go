package deepseek_v3

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCacher_Get(t *testing.T) {
	mockStore := NewMockStore()
	cacher := NewCacher(mockStore)

	t.Run("cache hit", func(t *testing.T) {
		// 初始化缓存数据
		mockStore.Set(context.Background(), "key1", "value1", 0)

		var result string
		found, err := cacher.Get(context.Background(), "key1", &result, nil, nil)
		assert.True(t, found)
		assert.NoError(t, err)
		assert.Equal(t, "value1", result)
	})

	t.Run("cache miss with fallback", func(t *testing.T) {
		fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
			return "fallback_value", true, nil
		}

		var result string
		found, err := cacher.Get(context.Background(), "key2", &result, fallback, &CacheOptions{TTL: time.Minute})
		assert.True(t, found)
		assert.NoError(t, err)
		assert.Equal(t, "fallback_value", result)
	})
}