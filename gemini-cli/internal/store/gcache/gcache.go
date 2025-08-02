package gcache

import (
    "context"
    "encoding/json"
    "reflect"
    "time"

    "github.com/bluele/gcache"
)

type gcacheStore struct {
    cache gcache.Cache
}

func NewGcacheStore(cache gcache.Cache) *gcacheStore {
    return &gcacheStore{cache: cache}
}

func (s *gcacheStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
    val, err := s.cache.Get(key)
    if err != nil {
        if err == gcache.KeyNotFoundError {
            return false, nil
        }
        return false, err
    }

    data, err := json.Marshal(val)
    if err != nil {
        return false, err
    }

    return true, json.Unmarshal(data, dst)
}

func (s *gcacheStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
    if len(keys) == 0 {
        return nil
    }

    dstMapVal := reflect.ValueOf(dstMap).Elem()
    elemType := dstMapVal.Type().Elem()

    for _, key := range keys {
        val, err := s.cache.Get(key)
        if err != nil {
            if err == gcache.KeyNotFoundError {
                continue
            }
            return err
        }

        data, err := json.Marshal(val)
        if err != nil {
            return err
        }

        dst := reflect.New(elemType).Interface()
        if err := json.Unmarshal(data, dst); err != nil {
            return err
        }
        dstMapVal.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(dst).Elem())
    }

    return nil
}

func (s *gcacheStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
    result := make(map[string]bool)
    for _, key := range keys {
        result[key] = s.cache.Has(key)
    }
    return result, nil
}

func (s *gcacheStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
    for key, val := range items {
        if ttl == 0 {
            if err := s.cache.Set(key, val); err != nil {
                return err
            }
        } else {
            if err := s.cache.SetWithExpire(key, val, ttl); err != nil {
                return err
            }
        }
    }
    return nil
}

func (s *gcacheStore) Del(ctx context.Context, keys ...string) (int64, error) {
    var count int64
    for _, key := range keys {
        if s.cache.Remove(key) {
            count++
        }
    }
    return count, nil
}
