package ristretto

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"time"

	"github.com/dgraph-io/ristretto/v2"
)

// Store 实现了store.Store接口，使用Ristretto作为后端存储
type Store struct {
	cache *ristretto.Cache[string, []byte]
	mux   sync.RWMutex // 用于保护ttlMap
	ttlMap map[string]time.Time // 存储键的过期时间
}

// Options 是Ristretto存储的配置选项
type Options struct {
	// Ristretto配置
	NumCounters      int64 // 计数器数量，应该是预期缓存项数量的10倍
	MaxCost          int64 // 缓存的最大成本，单位字节
	BufferItems      int64 // 缓存的缓冲区大小
	Metrics          bool  // 是否启用指标收集
	IgnoreInternalCost bool // 是否忽略内部成本计算

	// 如果提供了已存在的Ristretto缓存，将优先使用它
	Cache *ristretto.Cache[string, []byte]
}

// New 创建一个新的Ristretto存储实例
func New(opts Options) (*Store, error) {
	var cache *ristretto.Cache[string, []byte]
	var err error

	if opts.Cache != nil {
		cache = opts.Cache
	} else {
		// 使用默认值
		if opts.NumCounters <= 0 {
			opts.NumCounters = 1e7 // 1000万
		}
		if opts.MaxCost <= 0 {
			opts.MaxCost = 1 << 30 // 1GB
		}
		if opts.BufferItems <= 0 {
			opts.BufferItems = 64 // 默认值
		}

		// 创建Ristretto缓存
		cache, err = ristretto.NewCache[string, []byte](&ristretto.Config[string, []byte]{
			NumCounters:        opts.NumCounters,
			MaxCost:            opts.MaxCost,
			BufferItems:        opts.BufferItems,
			Metrics:            opts.Metrics,
			IgnoreInternalCost: opts.IgnoreInternalCost,
		})

		if err != nil {
			return nil, err
		}
	}

	return &Store{
		cache:  cache,
		ttlMap: make(map[string]time.Time),
	}, nil
}

// Get 从Ristretto获取键对应的值，并将其解析到value中
func (s *Store) Get(ctx context.Context, key string, value interface{}) (bool, error) {
	// 检查键是否过期
	if s.isExpired(key) {
		// 如果过期，删除键并返回未找到
		s.cache.Del(key)
		s.removeExpiration(key)
		return false, nil
	}

	// 从缓存获取值
	data, found := s.cache.Get(key)
	if !found || data == nil {
		return false, nil
	}

	// 解析JSON数据到目标值
	if err := json.Unmarshal(data, value); err != nil {
		return false, err
	}

	return true, nil
}

// MGet 从Ristretto批量获取多个键的值，并将结果解析到values中
func (s *Store) MGet(ctx context.Context, keys []string, values interface{}) error {
	// 检查values是否为map类型
	valuesVal := reflect.ValueOf(values)
	if valuesVal.Kind() != reflect.Ptr || valuesVal.Elem().Kind() != reflect.Map {
		return errors.New("values must be a pointer to a map")
	}

	// 获取map的类型信息
	mapVal := valuesVal.Elem()
	mapType := mapVal.Type()
	if mapType.Key().Kind() != reflect.String {
		return errors.New("map key must be string")
	}

	// 如果没有键，直接返回
	if len(keys) == 0 {
		return nil
	}

	// 批量获取键值
	for _, key := range keys {
		// 检查键是否过期
		if s.isExpired(key) {
			s.cache.Del(key)
			s.removeExpiration(key)
			continue
		}

		// 从缓存获取值
		data, found := s.cache.Get(key)
		if !found || data == nil {
			continue
		}

		// 创建一个新的目标值实例
		valType := mapType.Elem()
		newVal := reflect.New(valType)

		// 解析JSON
		if err := json.Unmarshal(data, newVal.Interface()); err != nil {
			continue // 跳过解析错误
		}

		// 设置到map中
		mapVal.SetMapIndex(reflect.ValueOf(key), newVal.Elem())
	}

	return nil
}

// Exists 检查多个键是否存在于Ristretto中
func (s *Store) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool, len(keys))

	// 初始化所有键为不存在
	for _, key := range keys {
		result[key] = false
	}

	// 如果没有键，直接返回
	if len(keys) == 0 {
		return result, nil
	}

	// 检查每个键是否存在
	for _, key := range keys {
		// 检查键是否过期
		if s.isExpired(key) {
			s.cache.Del(key)
			s.removeExpiration(key)
			continue
		}

		// 检查键是否存在
		_, found := s.cache.Get(key)
		result[key] = found
	}

	return result, nil
}

// MSet 批量设置多个键值对到Ristretto
func (s *Store) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	// 计算过期时间
	var expireAt time.Time
	if ttl > 0 {
		expireAt = time.Now().Add(ttl)
	}

	for key, value := range items {
		// 序列化值
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}

		// 设置到缓存
		s.cache.Set(key, data, 1) // 成本设为1，实际成本由Ristretto内部计算

		// 如果有TTL，记录过期时间
		if ttl > 0 {
			s.setExpiration(key, expireAt)
		} else {
			// 如果没有TTL，移除之前可能存在的过期时间
			s.removeExpiration(key)
		}
	}

	// 等待写入完成
	s.cache.Wait()

	return nil
}

// Del 从Ristretto中删除一个或多个键
func (s *Store) Del(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	var deleted int64

	for _, key := range keys {
		// 检查键是否存在
		_, found := s.cache.Get(key)
		if found {
			// 删除键
			s.cache.Del(key)
			// 移除过期时间
			s.removeExpiration(key)
			deleted++
		}
	}

	return deleted, nil
}

// Close 关闭Ristretto缓存
func (s *Store) Close() error {
	s.cache.Close()
	return nil
}

// isExpired 检查键是否已过期
func (s *Store) isExpired(key string) bool {
	s.mux.RLock()
	defer s.mux.RUnlock()

	expireAt, exists := s.ttlMap[key]
	if !exists {
		return false
	}

	return time.Now().After(expireAt)
}

// setExpiration 设置键的过期时间
func (s *Store) setExpiration(key string, expireAt time.Time) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.ttlMap[key] = expireAt
}

// removeExpiration 移除键的过期时间
func (s *Store) removeExpiration(key string) {
	s.mux.Lock()
	defer s.mux.Unlock()

	delete(s.ttlMap, key)
}