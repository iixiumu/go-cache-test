package cacher

import (
    "context"
    "reflect"
    "testing"
    "time"

    miniredis "github.com/alicebob/miniredis/v2"
    goredis "github.com/redis/go-redis/v9"
    dristretto "github.com/dgraph-io/ristretto/v2"

    ristrettoStore "go-cache/cacher/store/ristretto"
    redisStore "go-cache/cacher/store/redis"
)

func TestCacher_Get_Set_Fallback_Redis(t *testing.T) {
    mr, _ := miniredis.Run()
    t.Cleanup(mr.Close)
    client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
    t.Cleanup(func() { _ = client.Close() })
    c := New(redisStore.NewRedisStore(client))

    ctx := context.Background()
    var got int
    hit, err := c.Get(ctx, "k1", &got, func(ctx context.Context, key string) (interface{}, bool, error) {
        return 123, true, nil
    }, &CacheOptions{TTL: 50 * time.Millisecond})
    if err != nil || !hit || got != 123 {
        t.Fatalf("unexpected first get: hit=%v got=%v err=%v", hit, got, err)
    }

    // Next get should hit cache
    got = 0
    hit, err = c.Get(ctx, "k1", &got, nil, nil)
    if err != nil || !hit || got != 123 {
        t.Fatalf("unexpected cache hit: hit=%v got=%v err=%v", hit, got, err)
    }

    // Expire and fallback again
    mr.FastForward(2 * time.Second)
    got = 0
    hit, err = c.Get(ctx, "k1", &got, func(ctx context.Context, key string) (interface{}, bool, error) {
        return 456, true, nil
    }, nil)
    if err != nil || !hit || got != 456 {
        t.Fatalf("unexpected after expire: hit=%v got=%v err=%v", hit, got, err)
    }
}

func newRistretto(t *testing.T) *ristrettoStore.RistrettoStore {
    t.Helper()
    cache, err := dristretto.NewCache[string, any](&dristretto.Config[string, any]{
        NumCounters: 1e4,
        MaxCost:     1 << 20,
        BufferItems: 64,
    })
    if err != nil {
        t.Fatalf("failed to create ristretto: %v", err)
    }
    t.Cleanup(cache.Close)
    return ristrettoStore.NewRistrettoStore(cache)
}

func TestCacher_MGet_And_Refresh(t *testing.T) {
    // Use in-memory store for simplicity
    rs := newRistretto(t)
    c := New(rs)

    ctx := context.Background()
    keys := []string{"a", "b", "c"}
    // First call, none in cache
    m := map[string]int{}
    err := c.MGet(ctx, keys, &m, func(ctx context.Context, keys []string) (map[string]interface{}, error) {
        out := make(map[string]interface{}, len(keys))
        for i, k := range keys {
            out[k] = i + 1
        }
        return out, nil
    }, &CacheOptions{TTL: 100 * time.Millisecond})
    if err != nil {
        t.Fatalf("MGet error: %v", err)
    }
    if len(m) != 3 || m["a"] != 1 || m["b"] != 2 || m["c"] != 3 {
        t.Fatalf("unexpected first MGet map: %+v", m)
    }

    // Second call: should be all cache hits
    m2 := map[string]int{}
    err = c.MGet(ctx, keys, &m2, nil, nil)
    if err != nil || !reflect.DeepEqual(m, m2) {
        t.Fatalf("unexpected second MGet: err=%v got=%+v", err, m2)
    }

    // Refresh with new values
    err = c.MRefresh(ctx, keys, &m2, func(ctx context.Context, keys []string) (map[string]interface{}, error) {
        out := make(map[string]interface{}, len(keys))
        for i, k := range keys {
            out[k] = (i + 1) * 10
        }
        return out, nil
    }, nil)
    if err != nil {
        t.Fatalf("MRefresh error: %v", err)
    }
    if m2["a"] != 10 || m2["b"] != 20 || m2["c"] != 30 {
        t.Fatalf("unexpected refreshed map: %+v", m2)
    }
}
