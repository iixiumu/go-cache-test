package cache

import (
    "context"
    "errors"
    "reflect"
    "strconv"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
)

type mockStore struct {
    data map[string]interface{}
}

func newMockStore() *mockStore {
    return &mockStore{data: make(map[string]interface{})}
}

func (s *mockStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
    val, ok := s.data[key]
    if !ok {
        return false, nil
    }
    reflect.ValueOf(dst).Elem().Set(reflect.ValueOf(val))
    return true, nil
}

func (s *mockStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
    dstMapVal := reflect.ValueOf(dstMap).Elem()
    for _, key := range keys {
        if val, ok := s.data[key]; ok {
            dstMapVal.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(val))
        }
    }
    return nil
}

func (s *mockStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
    res := make(map[string]bool)
    for _, key := range keys {
        _, ok := s.data[key]
        res[key] = ok
    }
    return res, nil
}

func (s *mockStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
    for key, val := range items {
        s.data[key] = val
    }
    return nil
}

func (s *mockStore) Del(ctx context.Context, keys ...string) (int64, error) {
    var count int64
    for _, key := range keys {
        if _, ok := s.data[key]; ok {
            delete(s.data, key)
            count++
        }
    }
    return count, nil
}

func TestCacher_Get(t *testing.T) {
    store := newMockStore()
    cacher := NewCacher(store)

    t.Run("cache hit", func(t *testing.T) {
        store.data["key1"] = "value1"
        var dst string
        found, err := cacher.Get(context.Background(), "key1", &dst, nil, nil)
        assert.NoError(t, err)
        assert.True(t, found)
        assert.Equal(t, "value1", dst)
    })

    t.Run("cache miss, fallback success", func(t *testing.T) {
        delete(store.data, "key2")
        var dst string
        found, err := cacher.Get(context.Background(), "key2", &dst, func(ctx context.Context, key string) (interface{}, bool, error) {
            return "value2", true, nil
        }, nil)
        assert.NoError(t, err)
        assert.True(t, found)
        assert.Equal(t, "value2", dst)
        val, ok := store.data["key2"]
        assert.True(t, ok)
        assert.Equal(t, "value2", val)
    })

    t.Run("cache miss, fallback not found", func(t *testing.T) {
        delete(store.data, "key3")
        var dst string
        found, err := cacher.Get(context.Background(), "key3", &dst, func(ctx context.Context, key string) (interface{}, bool, error) {
            return nil, false, nil
        }, nil)
        assert.NoError(t, err)
        assert.False(t, found)
        assert.Empty(t, dst)
    })

    t.Run("fallback error", func(t *testing.T) {
        delete(store.data, "key4")
        var dst string
        fallbackErr := errors.New("fallback error")
        found, err := cacher.Get(context.Background(), "key4", &dst, func(ctx context.Context, key string) (interface{}, bool, error) {
            return nil, false, fallbackErr
        }, nil)
        assert.ErrorIs(t, err, fallbackErr)
        assert.False(t, found)
    })
}

func TestCacher_MGet(t *testing.T) {
    store := newMockStore()
    cacher := NewCacher(store)

    t.Run("all keys hit", func(t *testing.T) {
        store.data["key1"] = "value1"
        store.data["key2"] = "value2"
        dstMap := make(map[string]string)
        err := cacher.MGet(context.Background(), []string{"key1", "key2"}, &dstMap, nil, nil)
        assert.NoError(t, err)
        assert.Equal(t, map[string]string{"key1": "value1", "key2": "value2"}, dstMap)
    })

    t.Run("partial hit, fallback success", func(t *testing.T) {
        store.data["key1"] = "value1"
        delete(store.data, "key2")
        dstMap := make(map[string]string)
        err := cacher.MGet(context.Background(), []string{"key1", "key2"}, &dstMap, func(ctx context.Context, keys []string) (map[string]interface{}, error) {
            assert.Equal(t, []string{"key2"}, keys)
            return map[string]interface{}{"key2": "value2"}, nil
        }, nil)
        assert.NoError(t, err)
        assert.Equal(t, map[string]string{"key1": "value1", "key2": "value2"}, dstMap)
        val, ok := store.data["key2"]
        assert.True(t, ok)
        assert.Equal(t, "value2", val)
    })

    t.Run("all miss, fallback success", func(t *testing.T) {
        delete(store.data, "key1")
        delete(store.data, "key2")
        dstMap := make(map[string]string)
        err := cacher.MGet(context.Background(), []string{"key1", "key2"}, &dstMap, func(ctx context.Context, keys []string) (map[string]interface{}, error) {
            return map[string]interface{}{"key1": "value1", "key2": "value2"}, nil
        }, nil)
        assert.NoError(t, err)
        assert.Equal(t, map[string]string{"key1": "value1", "key2": "value2"}, dstMap)
    })

    t.Run("fallback error", func(t *testing.T) {
        delete(store.data, "key1")
        fallbackErr := errors.New("fallback error")
        dstMap := make(map[string]string)
        err := cacher.MGet(context.Background(), []string{"key1"}, &dstMap, func(ctx context.Context, keys []string) (map[string]interface{}, error) {
            return nil, fallbackErr
        }, nil)
        assert.ErrorIs(t, err, fallbackErr)
    })
}

func TestCacher_MDelete(t *testing.T) {
    store := newMockStore()
    cacher := NewCacher(store)
    store.data["key1"] = "value1"
    store.data["key2"] = "value2"

    deleted, err := cacher.MDelete(context.Background(), []string{"key1", "key3"})
    assert.NoError(t, err)
    assert.Equal(t, int64(1), deleted)
    _, ok := store.data["key1"]
    assert.False(t, ok)
}

func TestCacher_MRefresh(t *testing.T) {
    store := newMockStore()
    cacher := NewCacher(store)

    dstMap := make(map[string]string)
    err := cacher.MRefresh(context.Background(), []string{"key1", "key2"}, &dstMap, func(ctx context.Context, keys []string) (map[string]interface{}, error) {
        res := make(map[string]interface{}, len(keys))
        for _, key := range keys {
            res[key] = "refreshed_" + key
        }
        return res, nil
    }, nil)

    assert.NoError(t, err)
    assert.Equal(t, map[string]string{"key1": "refreshed_key1", "key2": "refreshed_key2"}, dstMap)
    val, ok := store.data["key1"]
    assert.True(t, ok)
    assert.Equal(t, "refreshed_key1", val)
}

func BenchmarkCacher_Get(b *testing.B) {
    store := newMockStore()
    cacher := NewCacher(store)
    cacher.Get(context.Background(), "key", new(string), func(ctx context.Context, key string) (interface{}, bool, error) {
        return "value", true, nil
    }, nil)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        key := "key" + strconv.Itoa(i)
        cacher.Get(context.Background(), key, new(string), func(ctx context.Context, key string) (interface{}, bool, error) {
            return "value", true, nil
        }, nil)
    }
}
