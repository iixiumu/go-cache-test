package ristretto

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"go-cache/cacher/store"
)

// cacheItem 包装缓存项，包含过期时间信息
type cacheItem struct {
	Value     interface{}
	ExpiresAt time.Time
}

// isExpired 检查缓存项是否过期
func (ci *cacheItem) isExpired() bool {
	return !ci.ExpiresAt.IsZero() && time.Now().After(ci.ExpiresAt)
}

// Store Ristretto实现的Store接口
type Store struct {
	cache *ristretto.Cache[string, *cacheItem]
	mutex sync.RWMutex
}

// NewStore 创建新的Ristretto Store实例
func NewStore() (*Store, error) {
	cache, err := ristretto.NewCache(&ristretto.Config[string, *cacheItem]{
		NumCounters: 1e7,     // 10M 计数器数量
		MaxCost:     1 << 30, // 1GB 最大内存使用
		BufferItems: 64,      // 缓冲区大小
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create ristretto cache: %w", err)
	}

	return &Store{
		cache: cache,
	}, nil
}

// Close 关闭缓存
func (s *Store) Close() {
	s.cache.Close()
}

// Get 从缓存获取单个值
func (s *Store) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	item, found := s.cache.Get(key)
	if !found {
		return false, nil
	}

	// 检查是否过期
	if item.isExpired() {
		s.cache.Del(key)
		return false, nil
	}

	// 由于是内存缓存，直接复制值，无需序列化
	if err := s.copyValue(item.Value, dst); err != nil {
		return false, fmt.Errorf("failed to copy value: %w", err)
	}

	return true, nil
}

// MGet 批量获取值到map中
func (s *Store) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	if len(keys) == 0 {
		return nil
	}

	// 验证dstMap是map指针
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr || dstMapValue.Elem().Kind() != reflect.Map {
		return fmt.Errorf("dstMap must be a pointer to map")
	}

	mapValue := dstMapValue.Elem()
	mapType := mapValue.Type()
	valueType := mapType.Elem()

	// 如果map为nil，初始化它
	if mapValue.IsNil() {
		mapValue.Set(reflect.MakeMap(mapType))
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, key := range keys {
		item, found := s.cache.Get(key)
		if !found {
			continue
		}

		// 检查是否过期
		if item.isExpired() {
			s.cache.Del(key)
			continue
		}

		// 创建值类型的新实例
		valuePtr := reflect.New(valueType)
		
		// 复制值
		if err := s.copyValue(item.Value, valuePtr.Interface()); err != nil {
			return fmt.Errorf("failed to copy value for key %s: %w", key, err)
		}

		// 设置到map中
		keyValue := reflect.ValueOf(key)
		mapValue.SetMapIndex(keyValue, valuePtr.Elem())
	}

	return nil
}

// Exists 批量检查键存在性
func (s *Store) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool)
	
	if len(keys) == 0 {
		return result, nil
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, key := range keys {
		item, found := s.cache.Get(key)
		if found && !item.isExpired() {
			result[key] = true
		} else {
			if found && item.isExpired() {
				s.cache.Del(key)
			}
			result[key] = false
		}
	}

	return result, nil
}

// MSet 批量设置键值对，支持TTL
func (s *Store) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	for key, value := range items {
		item := &cacheItem{
			Value:     value,
			ExpiresAt: expiresAt,
		}

		// 使用值的大小作为cost，对于简单类型使用1
		cost := int64(1)
		if ttl > 0 {
			// 对于有TTL的项，设置更高的cost以避免被过早淘汰
			cost = 2
		}

		success := s.cache.SetWithTTL(key, item, cost, ttl)
		if !success {
			return fmt.Errorf("failed to set key %s in cache", key)
		}
	}

	// 等待写入完成
	s.cache.Wait()
	return nil
}

// Del 删除指定键
func (s *Store) Del(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	var deletedCount int64
	for _, key := range keys {
		_, found := s.cache.Get(key)
		if found {
			s.cache.Del(key)
			deletedCount++
		}
	}

	return deletedCount, nil
}

// copyValue 复制值，处理不同类型的复制逻辑
func (s *Store) copyValue(src, dst interface{}) error {
	// 处理源值为nil的情况
	if src == nil {
		dstValue := reflect.ValueOf(dst)
		if dstValue.Kind() != reflect.Ptr {
			return fmt.Errorf("dst must be a pointer")
		}
		dstElem := dstValue.Elem()
		// 设置为零值
		dstElem.Set(reflect.Zero(dstElem.Type()))
		return nil
	}

	srcValue := reflect.ValueOf(src)
	dstValue := reflect.ValueOf(dst)

	// dst必须是指针
	if dstValue.Kind() != reflect.Ptr {
		return fmt.Errorf("dst must be a pointer")
	}

	dstElem := dstValue.Elem()
	
	// 如果类型相同，直接设置
	if srcValue.Type().AssignableTo(dstElem.Type()) {
		dstElem.Set(srcValue)
		return nil
	}

	// 如果类型不同但可以转换
	if srcValue.Type().ConvertibleTo(dstElem.Type()) {
		dstElem.Set(srcValue.Convert(dstElem.Type()))
		return nil
	}

	return fmt.Errorf("cannot copy value of type %T to %T", src, dst)
}

// 确保Store实现了store.Store接口
var _ store.Store = (*Store)(nil)
