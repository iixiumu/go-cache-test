package cacher

import (
    "context"
    "errors"
    "reflect"
    "time"

    "go-cache/cacher/store"
)

// Ensure implementation satisfies interface
var _ Cacher = (*cacherImpl)(nil)

type cacherImpl struct {
    store store.Store
}

func New(store store.Store) Cacher {
    return &cacherImpl{store: store}
}

func (c *cacherImpl) Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error) {
    // Try cache first
    hit, err := c.store.Get(ctx, key, dst)
    if err != nil {
        return false, err
    }
    if hit {
        return true, nil
    }
    if fallback == nil {
        return false, nil
    }
    // Fallback
    val, found, err := fallback(ctx, key)
    if err != nil {
        return false, err
    }
    if !found {
        return false, nil
    }
    // Assign to dst via reflection
    if err := assignTo(dst, val); err != nil {
        return false, err
    }
    // Save back to store
    ttl := durationFromOpts(opts)
    if err := c.store.MSet(ctx, map[string]interface{}{key: val}, ttl); err != nil {
        return true, err
    }
    return true, nil
}

func (c *cacherImpl) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
    // First attempt to fill from cache
    if err := c.store.MGet(ctx, keys, dstMap); err != nil {
        return err
    }
    // Determine missing keys by comparing dstMap keys
    missing := filterMissingKeys(keys, dstMap)
    if len(missing) == 0 {
        return nil
    }
    if fallback == nil {
        return nil
    }
    // Fetch missing
    fetched, err := fallback(ctx, missing)
    if err != nil {
        return err
    }
    if len(fetched) == 0 {
        return nil
    }
    // Merge into dstMap using reflection
    if err := mergeIntoMap(dstMap, fetched); err != nil {
        return err
    }
    // Save fetched into store
    ttl := durationFromOpts(opts)
    if err := c.store.MSet(ctx, fetched, ttl); err != nil {
        return err
    }
    return nil
}

func (c *cacherImpl) MDelete(ctx context.Context, keys []string) (int64, error) {
    return c.store.Del(ctx, keys...)
}

func (c *cacherImpl) MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
    if fallback == nil {
        return errors.New("fallback is required for refresh")
    }
    fetched, err := fallback(ctx, keys)
    if err != nil {
        return err
    }
    if err := replaceMap(dstMap, fetched); err != nil {
        return err
    }
    ttl := durationFromOpts(opts)
    if err := c.store.MSet(ctx, fetched, ttl); err != nil {
        return err
    }
    return nil
}

func durationFromOpts(opts *CacheOptions) (ttl time.Duration) {
    if opts == nil {
        return 0
    }
    return opts.TTL
}

// assignTo assigns value to dst pointer using reflection.
func assignTo(dst interface{}, value any) error {
    rv := reflect.ValueOf(dst)
    if rv.Kind() != reflect.Ptr || rv.IsNil() {
        return errors.New("dst must be a non-nil pointer")
    }
    ev := rv.Elem()
    val := reflect.ValueOf(value)
    if !val.Type().AssignableTo(ev.Type()) && val.Type().ConvertibleTo(ev.Type()) {
        val = val.Convert(ev.Type())
    }
    if !val.Type().AssignableTo(ev.Type()) {
        return errors.New("value type not assignable to destination")
    }
    ev.Set(val)
    return nil
}

// filterMissingKeys returns those keys not present in dstMap (which is *map[string]T)
func filterMissingKeys(keys []string, dstMap interface{}) []string {
    rv := reflect.ValueOf(dstMap)
    if rv.Kind() != reflect.Ptr || rv.IsNil() {
        return keys
    }
    mv := rv.Elem()
    if mv.Kind() != reflect.Map {
        return keys
    }
    missing := make([]string, 0, len(keys))
    for _, k := range keys {
        if !mv.MapIndex(reflect.ValueOf(k)).IsValid() {
            missing = append(missing, k)
        }
    }
    return missing
}

// mergeIntoMap adds entries from src (map[string]interface{}) into dstMap (*map[string]T) with conversion
func mergeIntoMap(dstMap interface{}, src map[string]interface{}) error {
    rv := reflect.ValueOf(dstMap)
    if rv.Kind() != reflect.Ptr || rv.IsNil() {
        return errors.New("dstMap must be a non-nil pointer to map[string]T")
    }
    mv := rv.Elem()
    if mv.Kind() != reflect.Map {
        return errors.New("dstMap must point to a map")
    }
    if mv.IsNil() {
        mv.Set(reflect.MakeMapWithSize(mv.Type(), len(src)))
    }
    if mv.Type().Key().Kind() != reflect.String {
        return errors.New("map key type must be string")
    }
    elemType := mv.Type().Elem()
    for k, v := range src {
        val := reflect.ValueOf(v)
        if !val.Type().AssignableTo(elemType) && val.Type().ConvertibleTo(elemType) {
            val = val.Convert(elemType)
        }
        if !val.Type().AssignableTo(elemType) {
            return errors.New("value type not assignable to map element type")
        }
        mv.SetMapIndex(reflect.ValueOf(k), val)
    }
    return nil
}

// replaceMap replaces the contents of dstMap with src
func replaceMap(dstMap interface{}, src map[string]interface{}) error {
    rv := reflect.ValueOf(dstMap)
    if rv.Kind() != reflect.Ptr || rv.IsNil() {
        return errors.New("dstMap must be a non-nil pointer to map[string]T")
    }
    mv := rv.Elem()
    if mv.Kind() != reflect.Map {
        return errors.New("dstMap must point to a map")
    }
    // Clear existing
    iter := mv.MapRange()
    for iter.Next() {
        mv.SetMapIndex(iter.Key(), reflect.Value{})
    }
    if mv.IsNil() {
        mv.Set(reflect.MakeMapWithSize(mv.Type(), len(src)))
    }
    if mv.Type().Key().Kind() != reflect.String {
        return errors.New("map key type must be string")
    }
    elemType := mv.Type().Elem()
    for k, v := range src {
        val := reflect.ValueOf(v)
        if !val.Type().AssignableTo(elemType) && val.Type().ConvertibleTo(elemType) {
            val = val.Convert(elemType)
        }
        if !val.Type().AssignableTo(elemType) {
            return errors.New("value type not assignable to map element type")
        }
        mv.SetMapIndex(reflect.ValueOf(k), val)
    }
    return nil
}
