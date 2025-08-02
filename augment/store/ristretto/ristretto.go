package ristretto

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/dgraph-io/ristretto"
)

// RistrettoStore Ristretto存储实现
type RistrettoStore struct {
	cache *ristretto.Cache
}

// NewRistrettoStore 创建新的Ristretto存储实例
func NewRistrettoStore(cache *ristretto.Cache) *RistrettoStore {
	return &RistrettoStore{
		cache: cache,
	}
}

// NewDefaultRistrettoStore 创建带默认配置的Ristretto存储实例
func NewDefaultRistrettoStore() (*RistrettoStore, error) {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // 10M counters
		MaxCost:     1 << 30, // 1GB
		BufferItems: 64,      // 64 items buffer
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create ristretto cache: %w", err)
	}

	return &RistrettoStore{
		cache: cache,
	}, nil
}

// cacheItem 缓存项，包含数据和过期时间
type cacheItem struct {
	Data      json.RawMessage `json:"data"`
	ExpiresAt int64           `json:"expires_at"` // Unix timestamp, 0表示永不过期
}

// isExpired 检查缓存项是否过期
func (item *cacheItem) isExpired() bool {
	if item.ExpiresAt == 0 {
		return false
	}
	return time.Now().Unix() > item.ExpiresAt
}

// Get 从Ristretto获取单个值
func (r *RistrettoStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	// 验证dst是指针
	if reflect.TypeOf(dst).Kind() != reflect.Ptr {
		return false, fmt.Errorf("dst must be a pointer")
	}

	value, found := r.cache.Get(key)
	if !found {
		return false, nil
	}

	item, ok := value.(*cacheItem)
	if !ok {
		// 清理无效数据
		r.cache.Del(key)
		return false, nil
	}

	// 检查是否过期
	if item.isExpired() {
		r.cache.Del(key)
		return false, nil
	}

	// 反序列化数据
	if err := json.Unmarshal(item.Data, dst); err != nil {
		return false, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return true, nil
}

// MGet 批量获取值到map中
func (r *RistrettoStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	// 验证dstMap是map指针
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr || dstMapValue.Elem().Kind() != reflect.Map {
		return fmt.Errorf("dstMap must be a pointer to map")
	}

	if len(keys) == 0 {
		return nil
	}

	mapValue := dstMapValue.Elem()
	mapType := mapValue.Type()
	valueType := mapType.Elem()

	// 确保map已初始化
	if mapValue.IsNil() {
		mapValue.Set(reflect.MakeMap(mapType))
	}

	// 逐个获取键值
	for _, key := range keys {
		value, found := r.cache.Get(key)
		if !found {
			continue
		}

		item, ok := value.(*cacheItem)
		if !ok {
			// 清理无效数据
			r.cache.Del(key)
			continue
		}

		// 检查是否过期
		if item.isExpired() {
			r.cache.Del(key)
			continue
		}

		// 创建目标类型的新实例
		valuePtr := reflect.New(valueType)
		if err := json.Unmarshal(item.Data, valuePtr.Interface()); err != nil {
			return fmt.Errorf("failed to unmarshal JSON for key %s: %w", key, err)
		}

		// 设置到map中
		mapValue.SetMapIndex(reflect.ValueOf(key), valuePtr.Elem())
	}

	return nil
}

// Exists 批量检查键存在性
func (r *RistrettoStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool)

	for _, key := range keys {
		value, found := r.cache.Get(key)
		if !found {
			result[key] = false
			continue
		}

		item, ok := value.(*cacheItem)
		if !ok {
			// 清理无效数据
			r.cache.Del(key)
			result[key] = false
			continue
		}

		// 检查是否过期
		if item.isExpired() {
			r.cache.Del(key)
			result[key] = false
			continue
		}

		result[key] = true
	}

	return result, nil
}

// MSet 批量设置键值对，支持TTL
func (r *RistrettoStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	var expiresAt int64
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl).Unix()
	}

	for key, value := range items {
		// 序列化数据
		jsonData, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON for key %s: %w", key, err)
		}

		item := &cacheItem{
			Data:      jsonData,
			ExpiresAt: expiresAt,
		}

		// 设置到缓存中，cost设为数据大小
		cost := int64(len(jsonData))
		if !r.cache.Set(key, item, cost) {
			// Ristretto的Set可能因为内存限制失败，但这不应该返回错误
			// 在实际应用中可能需要日志记录
		}
	}

	// 等待所有设置操作完成
	r.cache.Wait()

	return nil
}

// Del 删除指定键
func (r *RistrettoStore) Del(ctx context.Context, keys ...string) (int64, error) {
	var deleted int64

	for _, key := range keys {
		// 检查键是否存在
		if _, found := r.cache.Get(key); found {
			r.cache.Del(key)
			deleted++
		}
	}

	return deleted, nil
}

// Close 关闭Ristretto缓存
func (r *RistrettoStore) Close() {
	r.cache.Close()
}
