package cacher

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"time"

	"go-cache/cacher/store"
)

// DefaultCacher 是Cacher接口的默认实现
type DefaultCacher[T any] struct {
	store       store.Store
	prefix      string
	defaultTTL  time.Duration
	refreshLock sync.Mutex // 用于保护刷新操作
}

// DefaultCacherOptions 是DefaultCacher的配置选项
type DefaultCacherOptions struct {
	Store      store.Store  // 底层存储实现
	Prefix     string       // 键前缀
	DefaultTTL time.Duration // 默认TTL
}

// NewDefaultCacher 创建一个新的DefaultCacher实例
func NewDefaultCacher[T any](opts DefaultCacherOptions) *DefaultCacher[T] {
	// 如果没有设置Store，使用panic，因为这是一个严重错误
	if opts.Store == nil {
		panic("store is required")
	}

	// 如果没有设置默认TTL，使用1小时
	if opts.DefaultTTL <= 0 {
		opts.DefaultTTL = time.Hour
	}

	return &DefaultCacher[T]{
		store:      opts.Store,
		prefix:     opts.Prefix,
		defaultTTL: opts.DefaultTTL,
	}
}

// formatKey 格式化缓存键
func (c *DefaultCacher[T]) formatKey(key string) string {
	if c.prefix == "" {
		return key
	}
	return c.prefix + ":" + key
}

// Get 获取单个缓存项，如果缓存未命中则使用fallback函数获取
func (c *DefaultCacher[T]) Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error) {
	// 格式化键
	formattedKey := c.formatKey(key)

	// 尝试从缓存获取
	found, err := c.store.Get(ctx, formattedKey, dst)
	if err != nil {
		return false, err
	}

	// 如果找到缓存项，直接返回
	if found {
		return true, nil
	}

	// 如果没有fallback函数，返回未找到和错误
	if fallback == nil {
		return false, errors.New("cache miss and no fallback provided")
	}

	// 调用fallback函数获取数据
	value, found, err := fallback(ctx, key)
	if err != nil {
		return false, err
	}

	if !found {
		return false, nil
	}

	// 将结果复制到dst
	dstVal := reflect.ValueOf(dst)
	if dstVal.Kind() != reflect.Ptr || dstVal.IsNil() {
		return false, errors.New("dst must be a non-nil pointer")
	}

	// 使用反射设置值
	valueVal := reflect.ValueOf(value)
	if !valueVal.Type().AssignableTo(dstVal.Elem().Type()) {
		return false, errors.New("value type does not match dst type")
	}
	dstVal.Elem().Set(valueVal)

	// 确定TTL
	ttl := c.defaultTTL
	if opts != nil && opts.TTL > 0 {
		ttl = opts.TTL
	}

	// 将结果存入缓存
	items := map[string]interface{}{
		formattedKey: value,
	}
	if err := c.store.MSet(ctx, items, ttl); err != nil {
		// 存储错误不影响返回结果
		// 可以考虑记录日志
	}

	return true, nil
}

// MGet 批量获取缓存项，对于缓存未命中的键使用batchFallback函数获取
func (c *DefaultCacher[T]) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) ([]string, error) {
	// 检查dstMap是否为map类型的指针
	dstMapVal := reflect.ValueOf(dstMap)
	if dstMapVal.Kind() != reflect.Ptr || dstMapVal.Elem().Kind() != reflect.Map {
		return nil, errors.New("dstMap must be a pointer to a map")
	}

	// 获取map的类型信息
	mapVal := dstMapVal.Elem()
	mapType := mapVal.Type()
	
	// 检查map的键类型是否为string
	if mapType.Key().Kind() != reflect.String {
		return nil, errors.New("map key must be string")
	}
	
	// 检查map的值类型是否与泛型类型T兼容
	valueType := reflect.TypeOf((*T)(nil)).Elem()
	if !mapType.Elem().AssignableTo(valueType) && !mapType.Elem().Elem().AssignableTo(valueType) {
		return nil, errors.New("map value type is not compatible with generic type T")
	}

	// 如果没有键，直接返回
	if len(keys) == 0 {
		return []string{}, nil
	}

	// 格式化所有键
	formattedKeys := make([]string, len(keys))
	formattedToOriginal := make(map[string]string, len(keys))
	for i, key := range keys {
		formattedKey := c.formatKey(key)
		formattedKeys[i] = formattedKey
		formattedToOriginal[formattedKey] = key
	}

	// 从缓存批量获取
cachedItems := make(map[string]interface{})
err := c.store.MGet(ctx, formattedKeys, &cachedItems)
if err != nil {
	return nil, err
}

	// 转换回原始键并添加到结果map
	for formattedKey, value := range cachedItems {
		if originalKey, ok := formattedToOriginal[formattedKey]; ok {
			// 使用反射设置map值
			valueVal := reflect.ValueOf(value)
			
			// 检查map的值类型是否为指针类型
			if mapType.Elem().Kind() == reflect.Ptr {
				// 如果map值是指针类型，创建一个新的指针并设置其指向的值
				ptrType := mapType.Elem()
				ptrVal := reflect.New(ptrType.Elem())
				
				// 将缓存中的值复制到指针指向的结构体
				if err := deepCopy(value, ptrVal.Interface()); err == nil {
					mapVal.SetMapIndex(reflect.ValueOf(originalKey), ptrVal)
				}
			} else if valueVal.Type().AssignableTo(mapType.Elem()) {
				// 直接设置值
				mapVal.SetMapIndex(reflect.ValueOf(originalKey), valueVal)
			}
		}
	}

	// 找出缓存命中和未命中的键
	hitKeys := make([]string, 0)
	missingKeys := make([]string, 0)
	for _, key := range keys {
		if mapVal.MapIndex(reflect.ValueOf(key)).IsValid() {
			hitKeys = append(hitKeys, key)
		} else {
			missingKeys = append(missingKeys, key)
		}
	}

	// 如果所有键都命中缓存，直接返回
	if len(missingKeys) == 0 {
		return hitKeys, nil
	}

	// 如果没有fallback函数，返回部分结果和错误
	if fallback == nil {
		return hitKeys, errors.New("some keys missed cache and no batch fallback provided")
	}

	// 调用fallback函数获取缓存未命中的数据
	missingValues, foundKeys, err := fallback(ctx, missingKeys)
	if err != nil {
		// 即使有错误，也返回已经从缓存获取的部分结果
		return hitKeys, err
	}

	// 将缓存未命中的数据添加到结果map
	for key, value := range missingValues {
		// 使用反射设置map值
		valueVal := reflect.ValueOf(value)
		
		// 检查map的值类型是否为指针类型
		if mapType.Elem().Kind() == reflect.Ptr {
			// 如果map值是指针类型，创建一个新的指针并设置其指向的值
			ptrType := mapType.Elem()
			ptrVal := reflect.New(ptrType.Elem())
			
			// 将fallback返回的值复制到指针指向的结构体
			if valueVal.Kind() == reflect.Struct {
				// 如果fallback返回的是结构体，直接设置到指针指向的结构体
				reflect.Indirect(ptrVal).Set(valueVal)
				mapVal.SetMapIndex(reflect.ValueOf(key), ptrVal)
			} else if err := deepCopy(value, ptrVal.Interface()); err == nil {
				// 如果是其他类型，尝试使用deepCopy
				mapVal.SetMapIndex(reflect.ValueOf(key), ptrVal)
			}
		} else if valueVal.Type().AssignableTo(mapType.Elem()) {
			// 直接设置值
			mapVal.SetMapIndex(reflect.ValueOf(key), valueVal)
		}
	}
	
	// 将foundKeys添加到hitKeys中
	hitKeys = append(hitKeys, foundKeys...)

	// 确定TTL
	ttl := c.defaultTTL
	if opts != nil && opts.TTL > 0 {
		ttl = opts.TTL
	}

	// 将缓存未命中的数据存入缓存
	if len(missingValues) > 0 {
		cacheItems := make(map[string]interface{}, len(missingValues))
		for key, value := range missingValues {
			cacheItems[c.formatKey(key)] = value
		}
		if err := c.store.MSet(ctx, cacheItems, ttl); err != nil {
			// 存储错误不影响返回结果
			// 可以考虑记录日志
		}
	}

	// 合并缓存命中的键和fallback找到的键
	allFoundKeys := append(hitKeys, foundKeys...)
	return allFoundKeys, nil
}

// MDelete 批量删除缓存项
func (c *DefaultCacher[T]) MDelete(ctx context.Context, keys []string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	// 格式化所有键
	formattedKeys := make([]string, len(keys))
	for i, key := range keys {
		formattedKeys[i] = c.formatKey(key)
	}

	// 批量删除
	return c.store.Del(ctx, formattedKeys...)
}

// MRefresh 批量刷新缓存项
func (c *DefaultCacher[T]) MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	c.refreshLock.Lock()
	defer c.refreshLock.Unlock()

	// 检查dstMap是否为map类型的指针
	dstMapVal := reflect.ValueOf(dstMap)
	if dstMapVal.Kind() != reflect.Ptr || dstMapVal.Elem().Kind() != reflect.Map {
		return errors.New("dstMap must be a pointer to a map")
	}

	// 获取map的类型信息
	mapVal := dstMapVal.Elem()
	mapType := mapVal.Type()
	
	// 检查map的键类型是否为string
	if mapType.Key().Kind() != reflect.String {
		return errors.New("map key must be string")
	}
	
	// 检查map的值类型是否与泛型类型T兼容
	valueType := reflect.TypeOf((*T)(nil)).Elem()
	if !mapType.Elem().AssignableTo(valueType) && !mapType.Elem().Elem().AssignableTo(valueType) {
		return errors.New("map value type is not compatible with generic type T")
	}

	if len(keys) == 0 {
		return nil
	}

	// 如果没有fallback函数，无法刷新
	if fallback == nil {
		return errors.New("batch fallback is required for refresh")
	}

	// 调用fallback函数获取最新数据
	freshValues, _, err := fallback(ctx, keys)
	if err != nil {
		return err
	}

	// 将最新数据添加到结果map
	for key, value := range freshValues {
		// 使用反射设置map值
		valueVal := reflect.ValueOf(value)
		
		// 检查map的值类型是否为指针类型
		if mapType.Elem().Kind() == reflect.Ptr {
			// 如果map值是指针类型，创建一个新的指针并设置其指向的值
			ptrType := mapType.Elem()
			ptrVal := reflect.New(ptrType.Elem())
			
			// 将缓存中的值复制到指针指向的结构体
			if err := deepCopy(value, ptrVal.Interface()); err == nil {
				mapVal.SetMapIndex(reflect.ValueOf(key), ptrVal)
			}
		} else if valueVal.Type().AssignableTo(mapType.Elem()) {
			// 直接设置值
			mapVal.SetMapIndex(reflect.ValueOf(key), valueVal)
		}
	}

	// 确定TTL
	ttl := c.defaultTTL
	if opts != nil && opts.TTL > 0 {
		ttl = opts.TTL
	}

	// 将最新数据存入缓存
	if len(freshValues) > 0 {
		cacheItems := make(map[string]interface{}, len(freshValues))
		for key, value := range freshValues {
			cacheItems[c.formatKey(key)] = value
		}
		if err := c.store.MSet(ctx, cacheItems, ttl); err != nil {
			return err
		}
	}

	return nil
}

// deepCopy 使用JSON序列化和反序列化实现深拷贝
func deepCopy(src, dst interface{}) error {
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}