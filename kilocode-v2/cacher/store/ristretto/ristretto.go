package ristretto

import (
	"context"
	"reflect"
	"sync"
	"time"

	"go-cache/cacher/store"

	"github.com/dgraph-io/ristretto/v2"
)

// RistrettoStore 实现Store接口的Ristretto内存存储
type RistrettoStore struct {
	cache *ristretto.Cache[string, interface{}]
	// 用于存储键的过期时间
	ttlMap sync.Map
}

// NewRistrettoStore 创建一个新的RistrettoStore实例
func NewRistrettoStore() (store.Store, error) {
	cache, err := ristretto.NewCache(&ristretto.Config[string, interface{}]{
		NumCounters: 1e7,     // number of keys to track frequency of (10M).
		MaxCost:     1 << 30, // maximum cost of cache (1GB).
		BufferItems: 64,      // number of keys per Get buffer.
	})
	if err != nil {
		return nil, err
	}

	return &RistrettoStore{
		cache:  cache,
		ttlMap: sync.Map{},
	}, nil
}

// Get 从Ristretto获取单个值
func (r *RistrettoStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	// 检查键是否过期
	if r.isExpired(key) {
		r.cache.Del(key)
		r.ttlMap.Delete(key)
		return false, nil
	}

	// 从缓存获取值
	value, found := r.cache.Get(key)
	if !found {
		return false, nil
	}

	// 使用反射将值复制到目标变量
	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() != reflect.Ptr {
		return false, &reflect.ValueError{Method: "Get", Kind: dstValue.Kind()}
	}

	if dstValue.IsNil() {
		return false, &reflect.ValueError{Method: "Get", Kind: dstValue.Kind()}
	}

	dstElem := dstValue.Elem()
	srcValue := reflect.ValueOf(value)

	// 如果类型匹配，直接设置值
	if srcValue.Type().AssignableTo(dstElem.Type()) {
		dstElem.Set(srcValue)
	} else {
		// 尝试转换类型
		if srcValue.Type().ConvertibleTo(dstElem.Type()) {
			dstElem.Set(srcValue.Convert(dstElem.Type()))
		} else {
			return false, &reflect.ValueError{Method: "Get", Kind: dstElem.Kind()}
		}
	}

	return true, nil
}

// MGet 批量从Ristretto获取值
func (r *RistrettoStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	// 使用反射来设置目标map
	dstValue, err := r.getMapPtrValue(dstMap)
	if err != nil {
		return err
	}

	// 清空目标map
	dstValue.Set(reflect.MakeMap(dstValue.Type()))

	// 批量获取值
	for _, key := range keys {
		// 检查键是否过期
		if r.isExpired(key) {
			r.cache.Del(key)
			r.ttlMap.Delete(key)
			continue
		}

		// 从缓存获取值
		value, found := r.cache.Get(key)
		if !found {
			continue
		}

		// 创建新元素
		elemType := dstValue.Type().Elem()
		elemValue := reflect.New(elemType).Elem()

		// 设置值
		srcValue := reflect.ValueOf(value)
		if srcValue.Type().AssignableTo(elemType) {
			elemValue.Set(srcValue)
		} else if srcValue.Type().ConvertibleTo(elemType) {
			elemValue.Set(srcValue.Convert(elemType))
		} else {
			continue
		}

		// 设置map值
		dstValue.SetMapIndex(reflect.ValueOf(key), elemValue)
	}

	return nil
}

// Exists 批量检查键在Ristretto中的存在性
func (r *RistrettoStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool)

	for _, key := range keys {
		// 检查键是否过期
		if r.isExpired(key) {
			r.cache.Del(key)
			r.ttlMap.Delete(key)
			result[key] = false
			continue
		}

		// 检查键是否存在
		_, found := r.cache.Get(key)
		result[key] = found
	}

	return result, nil
}

// MSet 批量设置键值对到Ristretto
func (r *RistrettoStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	// 计算过期时间
	var expireAt *time.Time
	if ttl > 0 {
		t := time.Now().Add(ttl)
		expireAt = &t
	}

	// 批量设置值
	for key, value := range items {
		// 设置过期时间
		if expireAt != nil {
			r.ttlMap.Store(key, *expireAt)
		} else {
			r.ttlMap.Delete(key)
		}

		// 设置缓存值，成本简单设为1
		r.cache.Set(key, value, 1)
	}

	// 等待值被写入缓存
	r.cache.Wait()

	return nil
}

// Del 从Ristretto删除指定键
func (r *RistrettoStore) Del(ctx context.Context, keys ...string) (int64, error) {
	count := int64(0)

	for _, key := range keys {
		// 删除缓存值
		r.cache.Del(key)
		count++

		// 删除过期时间
		r.ttlMap.Delete(key)
	}

	return count, nil
}

// getMapPtrValue 获取map指针的reflect.Value
func (r *RistrettoStore) getMapPtrValue(mapPtr interface{}) (reflect.Value, error) {
	v := reflect.ValueOf(mapPtr)
	if v.Kind() != reflect.Ptr {
		return reflect.Value{}, &reflect.ValueError{Method: "getMapPtrValue", Kind: v.Kind()}
	}
	if v.Elem().Kind() != reflect.Map {
		return reflect.Value{}, &reflect.ValueError{Method: "getMapPtrValue", Kind: v.Elem().Kind()}
	}
	return v.Elem(), nil
}

// isExpired 检查键是否过期
func (r *RistrettoStore) isExpired(key string) bool {
	expireAt, ok := r.ttlMap.Load(key)
	if !ok {
		return false
	}

	if expireTime, ok := expireAt.(time.Time); ok {
		return time.Now().After(expireTime)
	}

	return false
}
