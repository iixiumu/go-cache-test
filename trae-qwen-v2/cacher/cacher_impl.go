package cacher

import (
	"context"
	"reflect"
	"time"

	"go-cache/cacher/store"
)

// CacherImpl Cacher接口的实现
type CacherImpl struct {
	store store.Store
}

// NewCacherImpl 创建新的Cacher实例
func NewCacherImpl(store store.Store) *CacherImpl {
	return &CacherImpl{store: store}
}

// Get 获取单个缓存项，缓存未命中时执行回退函数并缓存结果
func (c *CacherImpl) Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error) {
	// 尝试从存储后端获取值
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

	// 将回退函数的结果存入缓存
	items := map[string]interface{}{key: value}
	ttl := c.getTTL(opts)
	if err := c.store.MSet(ctx, items, ttl); err != nil {
		return false, err
	}

	// 将值复制到dst
	c.copyValue(dst, value)

	return true, nil
}

// MGet 批量获取缓存项，支持部分命中和批量回退
func (c *CacherImpl) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	// 尝试从存储后端批量获取值
	if err := c.store.MGet(ctx, keys, dstMap); err != nil {
		return err
	}

	// 检查哪些键未命中
	missedKeys := c.getMissedKeys(keys, dstMap)
	if len(missedKeys) == 0 {
		return nil
	}

	// 对未命中的键执行批量回退函数
	fallbackValues, err := fallback(ctx, missedKeys)
	if err != nil {
		return err
	}

	// 将回退函数的结果存入缓存
	ttl := c.getTTL(opts)
	if err := c.store.MSet(ctx, fallbackValues, ttl); err != nil {
		return err
	}

	// 将回退函数的结果合并到dstMap中
	c.mergeValues(dstMap, fallbackValues)

	return nil
}

// MDelete 批量清除缓存项
func (c *CacherImpl) MDelete(ctx context.Context, keys []string) (int64, error) {
	return c.store.Del(ctx, keys...)
}

// MRefresh 批量强制刷新缓存项
func (c *CacherImpl) MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	// 执行批量回退函数获取最新值
	values, err := fallback(ctx, keys)
	if err != nil {
		return err
	}

	// 将最新值存入缓存
	ttl := c.getTTL(opts)
	if err := c.store.MSet(ctx, values, ttl); err != nil {
		return err
	}

	// 将值复制到dstMap
	c.copyMap(dstMap, values)

	return nil
}

// getTTL 获取TTL值
func (c *CacherImpl) getTTL(opts *CacheOptions) time.Duration {
	if opts != nil {
		return opts.TTL
	}
	return 0
}

// getMissedKeys 获取未命中的键
func (c *CacherImpl) getMissedKeys(keys []string, dstMap interface{}) []string {
	// 获取dstMap的反射值
	mapValue := reflect.ValueOf(dstMap)
	if mapValue.Kind() == reflect.Ptr {
		mapValue = mapValue.Elem()
	}
	if mapValue.Kind() != reflect.Map {
		return keys
	}

	// 创建一个键的集合用于快速查找
	keySet := make(map[string]bool)
	for _, key := range keys {
		keySet[key] = true
	}

	// 遍历dstMap中的键，从keySet中移除已命中的键
	for _, mapKey := range mapValue.MapKeys() {
		key := mapKey.String()
		delete(keySet, key)
	}

	// 返回未命中的键
	missedKeys := make([]string, 0, len(keySet))
	for key := range keySet {
		missedKeys = append(missedKeys, key)
	}

	return missedKeys
}

// mergeValues 将回退函数的结果合并到dstMap中
func (c *CacherImpl) mergeValues(dstMap interface{}, values map[string]interface{}) {
	// 获取dstMap的反射值
	mapValue := reflect.ValueOf(dstMap)
	if mapValue.Kind() == reflect.Ptr {
		mapValue = mapValue.Elem()
	}
	if mapValue.Kind() != reflect.Map {
		return
	}

	// 将values中的键值对合并到dstMap中
	for key, value := range values {
		keyValue := reflect.ValueOf(key)
		valueValue := reflect.ValueOf(value)
		mapValue.SetMapIndex(keyValue, valueValue)
	}
}

// copyValue 将值复制到dst
func (c *CacherImpl) copyValue(dst interface{}, value interface{}) {
	// 获取dst的反射值
	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() != reflect.Ptr {
		return
	}
	dstValue = dstValue.Elem()

	// 获取value的反射值
	valueValue := reflect.ValueOf(value)

	// 将value的值设置到dst中
	if dstValue.CanSet() && dstValue.Type().AssignableTo(valueValue.Type()) {
		dstValue.Set(valueValue)
	}
}

// copyMap 将map复制到dstMap
func (c *CacherImpl) copyMap(dstMap interface{}, srcMap map[string]interface{}) {
	// 获取dstMap的反射值
	mapValue := reflect.ValueOf(dstMap)
	if mapValue.Kind() == reflect.Ptr {
		mapValue = mapValue.Elem()
	}
	if mapValue.Kind() != reflect.Map {
		return
	}

	// 将srcMap中的键值对复制到dstMap中
	for key, value := range srcMap {
		keyValue := reflect.ValueOf(key)
		valueValue := reflect.ValueOf(value)
		mapValue.SetMapIndex(keyValue, valueValue)
	}
}

// 确保CacherImpl实现了Cacher接口
var _ Cacher = (*CacherImpl)(nil)