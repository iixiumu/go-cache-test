package cache

import (
	"context"
	"reflect"
	"time"
)

// cacher implements the Cacher interface
type cacher struct {
	store Store
}

// NewCacher creates a new Cacher instance with the given Store
func NewCacher(store Store) Cacher {
	return &cacher{
		store: store,
	}
}

// Get 获取单个缓存项，缓存未命中时执行回退函数并缓存结果
func (c *cacher) Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error) {
	// 首先尝试从缓存获取
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
	
	// 将回退函数获取的值写入缓存
	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}
	
	items := map[string]interface{}{key: value}
	if err := c.store.MSet(ctx, items, ttl); err != nil {
		// 缓存写入失败不影响返回结果，只记录错误
		// 这里可以添加日志记录
	}
	
	// 将值复制到目标变量
	if err := copyValue(value, dst); err != nil {
		return false, err
	}
	
	return true, nil
}

// MGet 批量获取缓存项，支持部分命中和批量回退
func (c *cacher) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	if len(keys) == 0 {
		return nil
	}
	
	// 首先尝试从缓存批量获取
	if err := c.store.MGet(ctx, keys, dstMap); err != nil {
		return err
	}
	
	// 检查哪些键未命中
	dstMapValue := reflect.ValueOf(dstMap).Elem()
	missedKeys := make([]string, 0)
	
	for _, key := range keys {
		if !dstMapValue.MapIndex(reflect.ValueOf(key)).IsValid() {
			missedKeys = append(missedKeys, key)
		}
	}
	
	// 如果所有键都命中或没有回退函数，直接返回
	if len(missedKeys) == 0 || fallback == nil {
		return nil
	}
	
	// 执行批量回退函数
	fallbackData, err := fallback(ctx, missedKeys)
	if err != nil {
		return err
	}
	
	if len(fallbackData) == 0 {
		return nil
	}
	
	// 将回退数据写入缓存
	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}
	
	if err := c.store.MSet(ctx, fallbackData, ttl); err != nil {
		// 缓存写入失败不影响返回结果
	}
	
	// 将回退数据合并到结果map中
	for key, value := range fallbackData {
		dstMapValue.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
	}
	
	return nil
}

// MDelete 批量清除缓存项
func (c *cacher) MDelete(ctx context.Context, keys []string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}
	return c.store.Del(ctx, keys...)
}

// MRefresh 批量强制刷新缓存项
func (c *cacher) MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	if len(keys) == 0 {
		return nil
	}
	
	// 先删除现有缓存
	_, _ = c.store.Del(ctx, keys...)
	
	// 如果没有回退函数，直接返回
	if fallback == nil {
		return nil
	}
	
	// 执行批量回退函数获取新数据
	fallbackData, err := fallback(ctx, keys)
	if err != nil {
		return err
	}
	
	// 将新数据写入缓存
	ttl := time.Duration(0)
	if opts != nil {
		ttl = opts.TTL
	}
	
	if err := c.store.MSet(ctx, fallbackData, ttl); err != nil {
		return err
	}
	
	// 将数据复制到目标map
	dstMapValue := reflect.ValueOf(dstMap).Elem()
	for key, value := range fallbackData {
		dstMapValue.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
	}
	
	return nil
}

// copyValue 使用反射将源值复制到目标变量
func copyValue(src, dst interface{}) error {
	srcValue := reflect.ValueOf(src)
	dstValue := reflect.ValueOf(dst)
	
	if dstValue.Kind() != reflect.Ptr {
		return ErrInvalidDestination
	}
	
	dstElem := dstValue.Elem()
	if !dstElem.CanSet() {
		return ErrInvalidDestination
	}
	
	// 如果类型匹配，直接设置
	if srcValue.Type().AssignableTo(dstElem.Type()) {
		dstElem.Set(srcValue)
		return nil
	}
	
	// 如果类型可转换，进行转换
	if srcValue.Type().ConvertibleTo(dstElem.Type()) {
		dstElem.Set(srcValue.Convert(dstElem.Type()))
		return nil
	}
	
	return ErrTypeMismatch
}