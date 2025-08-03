package cache

import (
    "context"
    "reflect"
    "time"
)

var _ Cacher = (*cacher)(nil)

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
    if err != nil {
        return false, err
    }
    if !found {
        return false, nil
    }

    ttl := time.Duration(0)
    if opts != nil {
        ttl = opts.TTL
    }

    if err := c.store.MSet(ctx, map[string]interface{}{key: val}, ttl); err != nil {
        return false, err
    }

    reflect.ValueOf(dst).Elem().Set(reflect.ValueOf(val))
    return true, nil
}

func (c *cacher) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
    if err := c.store.MGet(ctx, keys, dstMap); err != nil {
        return err
    }

    dstMapVal := reflect.ValueOf(dstMap).Elem()
    mapType := dstMapVal.Type()
    if mapType.Kind() != reflect.Map {
        panic("dstMap must be a map")
    }

    var missedKeys []string
    for _, key := range keys {
        if !dstMapVal.MapIndex(reflect.ValueOf(key)).IsValid() {
            missedKeys = append(missedKeys, key)
        }
    }

    if len(missedKeys) == 0 {
        return nil
    }

    fallbackData, err := fallback(ctx, missedKeys)
    if err != nil {
        return err
    }

    if len(fallbackData) > 0 {
        ttl := time.Duration(0)
        if opts != nil {
            ttl = opts.TTL
        }
        if err := c.store.MSet(ctx, fallbackData, ttl); err != nil {
            return err
        }

        for key, val := range fallbackData {
            dstMapVal.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(val))
        }
    }

    return nil
}

func (c *cacher) MDelete(ctx context.Context, keys []string) (int64, error) {
    return c.store.Del(ctx, keys...)
}

func (c *cacher) MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
    fallbackData, err := fallback(ctx, keys)
    if err != nil {
        return err
    }

    if len(fallbackData) > 0 {
        ttl := time.Duration(0)
        if opts != nil {
            ttl = opts.TTL
        }
        if err := c.store.MSet(ctx, fallbackData, ttl); err != nil {
            return err
        }

        dstMapVal := reflect.ValueOf(dstMap).Elem()
        for key, val := range fallbackData {
            dstMapVal.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(val))
        }
    }

    return nil
}
