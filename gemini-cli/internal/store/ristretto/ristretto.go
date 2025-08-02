package ristretto

import (
    "bytes"
    "context"
    "encoding/json"
    "reflect"
    "time"

    "github.com/dgraph-io/ristretto"
)

type ristrettoStore struct {
    cache *ristretto.Cache
}

func NewRistrettoStore(cache *ristretto.Cache) *ristrettoStore {
    return &ristrettoStore{cache: cache}
}

func (s *ristrettoStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
    val, found := s.cache.Get(key)
    if !found {
        return false, nil
    }

    data, err := json.Marshal(val)
    if err != nil {
        return false, err
    }

    return true, json.Unmarshal(data, dst)
}

func (s *ristrettoStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
    if len(keys) == 0 {
        return nil
    }

    dstMapVal := reflect.ValueOf(dstMap).Elem()
    elemType := dstMapVal.Type().Elem()

    for _, key := range keys {
        val, found := s.cache.Get(key)
        if !found {
            continue
        }

        data, err := json.Marshal(val)
        if err != nil {
            return err
        }

        dst := reflect.New(elemType)
        decoder := json.NewDecoder(bytes.NewReader(data))
        decoder.UseNumber()
        if err := decoder.Decode(dst.Interface()); err != nil {
            return err
        }

        dstMapVal.SetMapIndex(reflect.ValueOf(key), dst.Elem())
    }

    return nil
}

func (s *ristrettoStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
    result := make(map[string]bool)
    for _, key := range keys {
        _, found := s.cache.Get(key)
        result[key] = found
    }
    return result, nil
}

func (s *ristrettoStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
    for key, val := range items {
        s.cache.SetWithTTL(key, val, 1, ttl)
    }
    s.cache.Wait()
    return nil
}

func (s *ristrettoStore) Del(ctx context.Context, keys ...string) (int64, error) {
    var count int64
    for _, key := range keys {
        s.cache.Del(key)
        count++
    }
    s.cache.Wait()
    return count, nil
}
