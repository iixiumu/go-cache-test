package cache

import (
	"context"
	"sync"
	"time"
)

// cacher Cacher接口的实现
type cacher struct {
	store Store
}

// NewCacher 创建新的Cacher实例
func NewCacher(store Store) Cacher {
	return &cacher{
		store: store,
	}
}

// Get 获取单个缓存项，缓存未命中时执行回退函数并缓存结果
func (c *cacher) Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error) {
	// 尝试从缓存获取
	found, err := c.store.Get(ctx, key, dst)
	if err != nil {
		return false, err
	}
	
	if found {
		return true, nil
	}
	
	// 缓存未命中，执行回退函数
	if fallback == nil {
		return false, nil
	}
	
	value, found, err := fallback(ctx, key)
	if err != nil {
		return false, err
	}
	
	if !found {
		return false, nil
	}
	
	// 将获取到的值设置到缓存
	ttl := time.Duration(0)
	if opts != nil && opts.TTL > 0 {
		ttl = opts.TTL
	}
	
	items := map[string]interface{}{
		key: value,
	}
	
	if err := c.store.MSet(ctx, items, ttl); err != nil {
		return false, err
	}
	
	// 将值反序列化到dst
	data, err := serializeValue(value)
	if err != nil {
		return false, err
	}
	
	if err := deserializeValue(data, dst); err != nil {
		return false, err
	}
	
	return true, nil
}

// MGet 批量获取缓存项，支持部分命中和批量回退
func (c *cacher) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	if len(keys) == 0 {
		return nil
	}
	
	// 从缓存获取
	cacheData := make(map[string][]byte)
	if err := c.store.MGet(ctx, keys, &cacheData); err != nil {
		return err
	}
	
	// 找出未命中的键
	missedKeys := make([]string, 0)
	for _, key := range keys {
		if _, exists := cacheData[key]; !exists {
			missedKeys = append(missedKeys, key)
		}
	}
	
	// 如果有未命中的键且提供了回退函数
	if len(missedKeys) > 0 && fallback != nil {
		fallbackData, err := fallback(ctx, missedKeys)
		if err != nil {
			return err
		}
		
		if len(fallbackData) > 0 {
			// 设置缓存选项
			ttl := time.Duration(0)
			if opts != nil && opts.TTL > 0 {
				ttl = opts.TTL
			}
			
			// 将回退数据写入缓存
			if err := c.store.MSet(ctx, fallbackData, ttl); err != nil {
				return err
			}
			
			// 合并数据
			for key, value := range fallbackData {
				data, err := serializeValue(value)
				if err != nil {
					continue
				}
				cacheData[key] = data
			}
		}
	}
	
	// 反序列化到目标map
	return deserializeMap(cacheData, dstMap)
}

// MDelete 批量清除缓存项
func (c *cacher) MDelete(ctx context.Context, keys []string) (int64, error) {
	return c.store.Del(ctx, keys...)
}

// MRefresh 批量强制刷新缓存项
func (c *cacher) MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	// 先删除现有缓存
	if _, err := c.store.Del(ctx, keys...); err != nil {
		return err
	}
	
	// 然后重新获取并缓存
	return c.MGet(ctx, keys, dstMap, fallback, opts)
}

// cacherWithLock 带锁的cacher实现，防止缓存击穿
type cacherWithLock struct {
	cacher
	locks *sync.Map
}

// NewCacherWithLock 创建带锁的Cacher实例
func NewCacherWithLock(store Store) Cacher {
	return &cacherWithLock{
		cacher: cacher{store: store},
		locks:  new(sync.Map),
	}
}

// Get 带锁的获取操作，防止缓存击穿
func (c *cacherWithLock) Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error) {
	// 尝试从缓存获取
	found, err := c.store.Get(ctx, key, dst)
	if err != nil {
		return false, err
	}
	
	if found {
		return true, nil
	}
	
	// 获取锁
	lockKey := "lock:" + key
	lock, _ := c.locks.LoadOrStore(lockKey, new(sync.Mutex))
	mu := lock.(*sync.Mutex)
	
	mu.Lock()
	defer mu.Unlock()
	
	// 双重检查，防止并发情况下重复计算
	found, err = c.store.Get(ctx, key, dst)
	if err != nil {
		return false, err
	}
	
	if found {
		return true, nil
	}
	
	// 执行回退逻辑
	return c.cacher.Get(ctx, key, dst, fallback, opts)
}