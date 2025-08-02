package deepseek_v3

import (
	"context"
	"encoding/json"
	"time"
)

type cacher struct {
	store Store
}

func NewCacher(store Store) Cacher {
	return &cacher{store: store}
}

func (c *cacher) Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error) {
	found, err := c.store.Get(ctx, key, dst)
	if err != nil {
		return false, err
	}
	if found {
		return true, nil
	}

	val, found, err := fallback(ctx, key)
	if err != nil || !found {
		return found, err
	}

	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}

	if err := c.store.Set(ctx, key, val, ttl); err != nil {
		return true, err
	}

	return json.Unmarshal([]byte(val.(string)), dst)
}

// 其他接口方法实现...