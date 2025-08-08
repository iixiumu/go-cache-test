package redis

import (
    "context"
    "encoding/json"
    "errors"
    "reflect"
    "time"

    goredis "github.com/redis/go-redis/v9"

    "go-cache/cacher/store"
)

// Ensure RedisStore implements store.Store
var _ store.Store = (*RedisStore)(nil)

type RedisStore struct {
    client *goredis.Client
}

func NewRedisStore(client *goredis.Client) *RedisStore {
    return &RedisStore{client: client}
}

// TestFastForward is used only by tests to advance time in miniredis.
// It is a no-op when not backed by miniredis but miniredis supports fast-forward via Do("TIME") in v2.
// We expose a dedicated helper that the shared test will downcast to.
func (r *RedisStore) TestFastForward(d time.Duration) {
    if d <= 0 {
        return
    }
    // Fall back to real sleep to let TTLs expire under miniredis
    time.Sleep(d)
}

func (r *RedisStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
    if dst == nil {
        return false, errors.New("dst must be non-nil pointer")
    }
    val, err := r.client.Get(ctx, key).Bytes()
    if err == goredis.Nil {
        return false, nil
    }
    if err != nil {
        return false, err
    }
    if len(val) == 0 {
        return false, nil
    }
    if err := json.Unmarshal(val, dst); err != nil {
        return false, err
    }
    return true, nil
}

func (r *RedisStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
    // Expect dstMap to be *map[string]T
    // We'll decode JSON values per key
    res, err := r.client.MGet(ctx, keys...).Result()
    if err != nil {
        return err
    }

    // Prepare map via reflection
    // We use a small helper to set entries
    if err := setTypedMapFromJSONResults(dstMap, keys, res); err != nil {
        return err
    }
    return nil
}

func (r *RedisStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
    results := make(map[string]bool, len(keys))
    if len(keys) == 0 {
        return results, nil
    }
    pipe := r.client.Pipeline()
    cmds := make([]*goredis.IntCmd, 0, len(keys))
    for _, k := range keys {
        cmds = append(cmds, pipe.Exists(ctx, k))
    }
    if _, err := pipe.Exec(ctx); err != nil && err != goredis.Nil {
        return nil, err
    }
    for i, k := range keys {
        n, err := cmds[i].Result()
        if err != nil && err != goredis.Nil {
            return nil, err
        }
        results[k] = n > 0
    }
    return results, nil
}

func (r *RedisStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
    if len(items) == 0 {
        return nil
    }
    pipe := r.client.Pipeline()
    for k, v := range items {
        b, err := json.Marshal(v)
        if err != nil {
            return err
        }
        if ttl > 0 {
            // Use millisecond precision expiration to support sub-second TTLs reliably
            pipe.Set(ctx, k, b, 0)
            pipe.PExpire(ctx, k, ttl)
        } else {
            pipe.Set(ctx, k, b, 0)
        }
    }
    _, err := pipe.Exec(ctx)
    return err
}

func (r *RedisStore) Del(ctx context.Context, keys ...string) (int64, error) {
    if len(keys) == 0 {
        return 0, nil
    }
    n, err := r.client.Del(ctx, keys...).Result()
    if err != nil {
        return 0, err
    }
    return n, nil
}

// setTypedMapFromJSONResults uses reflection to set entries in *map[string]T from
// a slice of MGET results where each hit is string/[]byte(json) and miss is nil.
// keys and results must be aligned.
func setTypedMapFromJSONResults(dstMap interface{}, keys []string, results []interface{}) error {
    rv := reflect.ValueOf(dstMap)
    if rv.Kind() != reflect.Ptr || rv.IsNil() {
        return errors.New("dstMap must be a non-nil pointer to map[string]T")
    }
    rv = rv.Elem()
    if rv.Kind() != reflect.Map {
        return errors.New("dstMap must point to a map")
    }
    keyType := rv.Type().Key()
    if keyType.Kind() != reflect.String {
        return errors.New("map key type must be string")
    }
    elemType := rv.Type().Elem()
    if rv.IsNil() {
        rv.Set(reflect.MakeMapWithSize(rv.Type(), len(keys)))
    }
    for i, k := range keys {
        raw := results[i]
        if raw == nil {
            continue
        }
        var data []byte
        switch v := raw.(type) {
        case string:
            data = []byte(v)
        case []byte:
            data = v
        default:
            // Attempt to marshal whatever it is to JSON first
            b, err := json.Marshal(v)
            if err != nil {
                return err
            }
            data = b
        }
        elemPtr := reflect.New(elemType)
        if err := json.Unmarshal(data, elemPtr.Interface()); err != nil {
            return err
        }
        rv.SetMapIndex(reflect.ValueOf(k), elemPtr.Elem())
    }
    return nil
}
