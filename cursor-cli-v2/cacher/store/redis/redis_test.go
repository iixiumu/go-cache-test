package redis

import (
    "testing"
    "time"

    miniredis "github.com/alicebob/miniredis/v2"
    goredis "github.com/redis/go-redis/v9"

    "go-cache/cacher/store"
    "go-cache/cacher/store/storetest"
)

type testStore struct {
    *RedisStore
    mr *miniredis.Miniredis
}

func (tst *testStore) TestFastForward(d time.Duration) { tst.mr.FastForward(d) }

func newTestRedisStore(t *testing.T) store.Store {
    t.Helper()
    mr, err := miniredis.Run()
    if err != nil {
        t.Fatalf("failed to start miniredis: %v", err)
    }
    t.Cleanup(mr.Close)
    client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
    t.Cleanup(func() { _ = client.Close() })
    return &testStore{RedisStore: NewRedisStore(client), mr: mr}
}

func TestRedisStore_Compliance(t *testing.T) {
    storetest.Run(t, func(t *testing.T) store.Store { return newTestRedisStore(t) })
}
