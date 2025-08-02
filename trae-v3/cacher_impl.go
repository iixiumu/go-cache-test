package deepseek

import (
	"context"
	"time"
)

// cacher 实现Cacher接口的缓存器
type cacher struct {
	store Store
}

// NewCacher 创建新的缓存器实例
func NewCacher(store Store) Cacher {
	return &cacher{
		store: store,
	}
}

func (c *cacher) Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error) {
	found, err := c.store.Get(ctx, key, dst)
	if err != nil {
		return false, err
	}
	if found {
		return true, nil
	}

	if fallback == nil {
		return false, nil
	}

	value, found, err := fallback(ctx, key)
	if err != nil || !found {
		return found, err
	}

	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}

	err = c.store.MSet(ctx, map[string]interface{}{key: value}, ttl)
	if err != nil {
		return true, err
	}

	// 再次尝试从缓存获取
	return c.store.Get(ctx, key, dst)
}

func (c *cacher) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	// 实现批量获取逻辑
	return nil
}

func (c *cacher) MDelete(ctx context.Context, keys []string) (int64, error) {
	return c.store.Del(ctx, keys...)
}

func (c *cacher) MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	// 先删除旧缓存
	_, err := c.MDelete(ctx, keys)
	if err != nil {
		return err
	}

	// 然后重新获取
	return c.MGet(ctx, keys, dstMap, fallback, opts)
}