package cache

import (
	"context"
	"fmt"
	"reflect"
	"time"
)

// cacherImpl 实现Cacher接口
type cacherImpl struct {
	store Store
}

// NewCacher 创建新的Cacher实例
func NewCacher(store Store) Cacher {
	return &cacherImpl{
		store: store,
	}
}

// Get 获取单个缓存项，缓存未命中时执行回退函数并缓存结果
func (c *cacherImpl) Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error) {
	// 先尝试从缓存获取
	found, err := c.store.Get(ctx, key, dst)
	if err != nil {
		return false, err
	}
	
	if found {
		return true, nil
	}
	
	// 缓存未命中，执行回退函数
	value, found, err := fallback(ctx, key)
	if err != nil {
		return false, err
	}
	
	if !found {
		return false, nil
	}
	
	// 将回退获取的值缓存起来
	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}
	
	// 使用MSet来存储这个值
	items := map[string]interface{}{
		key: value,
	}
	
	err = c.store.MSet(ctx, items, ttl)
	if err != nil {
		return false, err
	}
	
	// 将值复制到dst
	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() == reflect.Ptr {
		dstValue.Elem().Set(reflect.ValueOf(value))
	}
	
	return true, nil
}

// MGet 批量获取缓存项，支持部分命中和批量回退
func (c *cacherImpl) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	// 先尝试从缓存获取所有值
	err := c.store.MGet(ctx, keys, dstMap)
	if err != nil {
		return err
	}
	
	// 检查哪些键在缓存中未命中
	mapValue := reflect.ValueOf(dstMap)
	if mapValue.Kind() != reflect.Ptr || mapValue.Elem().Kind() != reflect.Map {
		return fmt.Errorf("dstMap must be a pointer to a map")
	}
	
	mapElem := mapValue.Elem()
	
	// 构建未命中的键列表
	missingKeys := make([]string, 0)
	for _, key := range keys {
		// Check if the key exists and has a non-nil value
		mapKey := reflect.ValueOf(key)
		if mapElem.MapIndex(mapKey).IsValid() && !mapElem.MapIndex(mapKey).IsNil() {
			continue // Key exists and has a value
		} else {
			missingKeys = append(missingKeys, key)
		}
	}
	
	// 如果所有键都已命中，返回成功
	if len(missingKeys) == 0 {
		return nil
	}
	
	// 对未命中的键执行批量回退
	values, err := fallback(ctx, missingKeys)
	if err != nil {
		return err
	}
	
	// 将回退获取的值添加到结果中，并缓存它们
	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}
	
	// 创建需要缓存的键值对映射
	cacheItems := make(map[string]interface{})
	for key, value := range values {
		cacheItems[key] = value
		
		// 更新结果map
		mapElem.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
	}
	
	// 缓存新获取的数据
	err = c.store.MSet(ctx, cacheItems, ttl)
	if err != nil {
		return err
	}
	
	return nil
}

// MDelete 批量清除缓存项
func (c *cacherImpl) MDelete(ctx context.Context, keys []string) (int64, error) {
	return c.store.Del(ctx, keys...)
}

// MRefresh 批量强制刷新缓存项
func (c *cacherImpl) MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	// 清除缓存项
	_, err := c.store.Del(ctx, keys...)
	if err != nil {
		return err
	}
	
	// 使用批量回退函数重新获取数据
	values, err := fallback(ctx, keys)
	if err != nil {
		return err
	}
	
	// 将回退获取的值缓存起来
	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}
	
	// 缓存新获取的数据
	err = c.store.MSet(ctx, values, ttl)
	if err != nil {
		return err
	}
	
	// 更新结果map
	mapValue := reflect.ValueOf(dstMap)
	if mapValue.Kind() != reflect.Ptr || mapValue.Elem().Kind() != reflect.Map {
		return fmt.Errorf("dstMap must be a pointer to a map")
	}
	
	mapElem := mapValue.Elem()
	
	// 将获取的值放入结果map
	for key, value := range values {
		mapElem.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
	}
	
	return nil
}