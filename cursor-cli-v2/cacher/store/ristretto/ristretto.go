package ristretto

import (
    "context"
    "errors"
    "reflect"
    "time"

    dristretto "github.com/dgraph-io/ristretto/v2"

    "go-cache/cacher/store"
)

// Ensure RistrettoStore implements store.Store
var _ store.Store = (*RistrettoStore)(nil)

type RistrettoStore struct {
    cache *dristretto.Cache[string, any]
}

func NewRistrettoStore(cache *dristretto.Cache[string, any]) *RistrettoStore {
    return &RistrettoStore{cache: cache}
}

func (r *RistrettoStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
    if dst == nil {
        return false, errors.New("dst must be non-nil pointer")
    }
    v, ok := r.cache.Get(key)
    if !ok {
        return false, nil
    }
    if err := assignTo(dst, v); err != nil {
        return false, err
    }
    return true, nil
}

func (r *RistrettoStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
    rv := reflect.ValueOf(dstMap)
    if rv.Kind() != reflect.Ptr || rv.IsNil() {
        return errors.New("dstMap must be a non-nil pointer to map[string]T")
    }
    mv := rv.Elem()
    if mv.Kind() != reflect.Map {
        return errors.New("dstMap must point to a map")
    }
    if mv.IsNil() {
        mv.Set(reflect.MakeMapWithSize(mv.Type(), len(keys)))
    }
    if mv.Type().Key().Kind() != reflect.String {
        return errors.New("map key type must be string")
    }
    elemType := mv.Type().Elem()
    for _, k := range keys {
        if v, ok := r.cache.Get(k); ok {
            val := reflect.ValueOf(v)
            // Do not perform conversions for in-memory store; require exact assignability
            if !val.Type().AssignableTo(elemType) {
                // As per requirement, in-memory store shouldn't serialize/deserialize.
                // So if types mismatch in a non-convertible way, return an error.
                return errors.New("value type not assignable to map element type")
            }
            mv.SetMapIndex(reflect.ValueOf(k), val)
        }
    }
    return nil
}

func (r *RistrettoStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
    res := make(map[string]bool, len(keys))
    for _, k := range keys {
        _, ok := r.cache.Get(k)
        res[k] = ok
    }
    return res, nil
}

func (r *RistrettoStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
    for k, v := range items {
        if ttl > 0 {
            r.cache.SetWithTTL(k, v, 1, ttl)
        } else {
            r.cache.Set(k, v, 1)
        }
    }
    // Wait for value to pass through buffers
    r.cache.Wait()
    return nil
}

func (r *RistrettoStore) Del(ctx context.Context, keys ...string) (int64, error) {
    var n int64
    for _, k := range keys {
        if _, ok := r.cache.Get(k); ok {
            r.cache.Del(k)
            n++
        } else {
            // still call Del to ensure no residue
            r.cache.Del(k)
        }
    }
    // Wait for deletes to flush
    r.cache.Wait()
    return n, nil
}

// assignTo assigns value to dst pointer using reflection.
func assignTo(dst interface{}, value any) error {
    rv := reflect.ValueOf(dst)
    if rv.Kind() != reflect.Ptr || rv.IsNil() {
        return errors.New("dst must be a non-nil pointer")
    }
    ev := rv.Elem()
    val := reflect.ValueOf(value)
    if !val.Type().AssignableTo(ev.Type()) {
        return errors.New("value type not assignable to destination")
    }
    ev.Set(val)
    return nil
}
