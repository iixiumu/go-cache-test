package redis

import (
    "context"
    "encoding/json"
    "errors"
    "reflect"
    "time"

    "github.com/redis/go-redis/v9"
)

type redisStore struct {
    client redis.Cmdable
}

func NewRedisStore(client redis.Cmdable) *redisStore {
    return &redisStore{client: client}
}

func (s *redisStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
    val, err := s.client.Get(ctx, key).Bytes()
    if err != nil {
        if errors.Is(err, redis.Nil) {
            return false, nil
        }
        return false, err
    }
    return true, json.Unmarshal(val, dst)
}

func (s *redisStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
    if len(keys) == 0 {
        return nil
    }

    vals, err := s.client.MGet(ctx, keys...).Result()
    if err != nil {
        return err
    }

    dstMapVal := reflect.ValueOf(dstMap).Elem()
    elemType := dstMapVal.Type().Elem()

    for i, v := range vals {
        if v == nil {
            continue
        }

        strVal, ok := v.(string)
        if !ok {
            continue
        }

        dst := reflect.New(elemType).Interface()
        if err := json.Unmarshal([]byte(strVal), dst); err != nil {
            return err
        }
        dstMapVal.SetMapIndex(reflect.ValueOf(keys[i]), reflect.ValueOf(dst).Elem())
    }

    return nil
}

func (s *redisStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
    if len(keys) == 0 {
        return map[string]bool{}, nil
    }

    pipe := s.client.Pipeline()
    cmds := make(map[string]*redis.IntCmd)
    for _, key := range keys {
        cmds[key] = pipe.Exists(ctx, key)
    }

    if _, err := pipe.Exec(ctx); err != nil {
        return nil, err
    }

    result := make(map[string]bool)
    for key, cmd := range cmds {
        result[key] = cmd.Val() > 0
    }

    return result, nil
}

func (s *redisStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
    if len(items) == 0 {
        return nil
    }

    pipe := s.client.Pipeline()
    for key, val := range items {
        data, err := json.Marshal(val)
        if err != nil {
            return err
        }
        pipe.Set(ctx, key, data, ttl)
    }

    _, err := pipe.Exec(ctx)
    return err
}

func (s *redisStore) Del(ctx context.Context, keys ...string) (int64, error) {
    if len(keys) == 0 {
        return 0, nil
    }
    return s.client.Del(ctx, keys...).Result()
}
