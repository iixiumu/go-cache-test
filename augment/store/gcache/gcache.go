package gcache

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/bluele/gcache"
)

// GCacheStore GCache存储实现
type GCacheStore struct {
	cache gcache.Cache
}

// NewGCacheStore 创建新的GCache存储实例
func NewGCacheStore(cache gcache.Cache) *GCacheStore {
	return &GCacheStore{
		cache: cache,
	}
}

// NewDefaultGCacheStore 创建带默认配置的GCache存储实例
func NewDefaultGCacheStore(size int) *GCacheStore {
	cache := gcache.New(size).
		LRU().
		Build()

	return &GCacheStore{
		cache: cache,
	}
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

// Get 从GCache获取单个值
func (g *GCacheStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	// 验证dst是指针
	if reflect.TypeOf(dst).Kind() != reflect.Ptr {
		return false, fmt.Errorf("dst must be a pointer")
	}

	value, err := g.cache.Get(key)
	if err != nil {
		if err == gcache.KeyNotFoundError {
			return false, nil
		}
		return false, fmt.Errorf("gcache get failed: %w", err)
	}

	item, ok := value.(*cacheItem)
	if !ok {
		// 清理无效数据
		g.cache.Remove(key)
		return false, nil
	}

	// 检查是否过期
	if item.isExpired() {
		g.cache.Remove(key)
		return false, nil
	}

	// 反序列化数据
	if err := json.Unmarshal(item.Data, dst); err != nil {
		return false, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return true, nil
}

// MGet 批量获取值到map中
func (g *GCacheStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
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
		value, err := g.cache.Get(key)
		if err != nil {
			if err == gcache.KeyNotFoundError {
				continue
			}
			return fmt.Errorf("gcache get failed for key %s: %w", key, err)
		}

		item, ok := value.(*cacheItem)
		if !ok {
			// 清理无效数据
			g.cache.Remove(key)
			continue
		}

		// 检查是否过期
		if item.isExpired() {
			g.cache.Remove(key)
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
func (g *GCacheStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool)

	for _, key := range keys {
		value, err := g.cache.Get(key)
		if err != nil {
			if err == gcache.KeyNotFoundError {
				result[key] = false
				continue
			}
			return nil, fmt.Errorf("gcache get failed for key %s: %w", key, err)
		}

		item, ok := value.(*cacheItem)
		if !ok {
			// 清理无效数据
			g.cache.Remove(key)
			result[key] = false
			continue
		}

		// 检查是否过期
		if item.isExpired() {
			g.cache.Remove(key)
			result[key] = false
			continue
		}

		result[key] = true
	}

	return result, nil
}

// MSet 批量设置键值对，支持TTL
func (g *GCacheStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
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

		// 设置到缓存中
		if ttl > 0 {
			err = g.cache.SetWithExpire(key, item, ttl)
		} else {
			err = g.cache.Set(key, item)
		}

		if err != nil {
			return fmt.Errorf("gcache set failed for key %s: %w", key, err)
		}
	}

	return nil
}

// Del 删除指定键
func (g *GCacheStore) Del(ctx context.Context, keys ...string) (int64, error) {
	var deleted int64

	for _, key := range keys {
		// 检查键是否存在
		if g.cache.Has(key) {
			if g.cache.Remove(key) {
				deleted++
			}
		}
	}

	return deleted, nil
}

// Purge 清空所有缓存
func (g *GCacheStore) Purge() {
	g.cache.Purge()
}

// Len 返回缓存中的项目数量
func (g *GCacheStore) Len() int {
	return g.cache.Len(false)
}
