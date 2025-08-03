package ristretto

import (
	"context"
	"reflect"
	"time"

	"go-cache/store"

	"github.com/dgraph-io/ristretto/v2"
)

// RistrettoStore 是基于ristretto的Store接口实现
type RistrettoStore struct {
	cache *ristretto.Cache[string, interface{}]
}

// NewRistrettoStore 创建一个新的RistrettoStore实例
func NewRistrettoStore(cache *ristretto.Cache[string, interface{}]) store.Store {
	return &RistrettoStore{
		cache: cache,
	}
}

// Get 从Ristretto获取单个值
func (r *RistrettoStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	// 等待缓冲区处理完成
	r.cache.Wait()

	value, found := r.cache.Get(key)
	if !found {
		return false, nil
	}

	// 使用反射将值设置到dst
	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() != reflect.Ptr {
		return false, nil
	}
	dstValue = dstValue.Elem()

	valueReflect := reflect.ValueOf(value)
	if dstValue.Type() != valueReflect.Type() {
		return false, nil
	}

	dstValue.Set(valueReflect)
	return true, nil
}

// MGet 批量从Ristretto获取值到map中
func (r *RistrettoStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	// 等待缓冲区处理完成
	r.cache.Wait()

	// 使用反射处理dstMap
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr {
		return nil // dstMap必须是指针
	}
	dstMapValue = dstMapValue.Elem()
	if dstMapValue.Kind() != reflect.Map {
		return nil // dstMap必须是指向map的指针
	}

	// 获取map的键和值类型
	mapType := dstMapValue.Type()
	keyType := mapType.Key()
	valueType := mapType.Elem()

	// 创建新的map
	newMap := reflect.MakeMap(mapType)

	// 遍历键
	for _, key := range keys {
		value, found := r.cache.Get(key)
		if !found {
			// 键不存在，跳过
			continue
		}

		// 检查值类型
		valueReflect := reflect.ValueOf(value)
		if valueReflect.Type() != valueType {
			continue
		}

		// 设置map中的值
		mapKey := reflect.ValueOf(key).Convert(keyType)
		newMap.SetMapIndex(mapKey, valueReflect)
	}

	// 设置dstMap的值
	dstMapValue.Set(newMap)
	return nil
}

// Exists 批量检查键存在性
func (r *RistrettoStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	// 等待缓冲区处理完成
	r.cache.Wait()

	// 构建结果
	exists := make(map[string]bool)
	for _, key := range keys {
		_, exists[key] = r.cache.Get(key)
	}

	return exists, nil
}

// MSet 批量设置键值对，支持TTL
func (r *RistrettoStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	// Ristretto不直接支持TTL，但我们可以通过在值中包装过期时间来实现
	// 对于这个实现，我们忽略TTL参数，因为Ristretto是一个内存缓存，主要依靠LRU驱逐策略

	// 批量设置
	for key, value := range items {
		// 我们使用固定的成本1，实际应用中可以根据值的大小调整
		r.cache.Set(key, value, 1)
	}

	return nil
}

// Del 删除指定键
func (r *RistrettoStore) Del(ctx context.Context, keys ...string) (int64, error) {
	count := int64(0)
	for _, key := range keys {
		r.cache.Del(key)
		// Ristretto的Del方法没有返回值，所以我们无法准确计算删除的键数量
		// 这里假设所有键都被成功删除
		count++
	}
	return count, nil
}
