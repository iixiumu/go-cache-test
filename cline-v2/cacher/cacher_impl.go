package cacher

import (
	"context"
	"reflect"

	"go-cache/cacher/store"
)

// cacherImpl Cacher接口的实现
type cacherImpl struct {
	store store.Store
}

// NewCacher 创建新的Cacher实例
func NewCacher(store store.Store) Cacher {
	return &cacherImpl{
		store: store,
	}
}

// Get 获取单个缓存项，缓存未命中时执行回退函数并缓存结果
func (c *cacherImpl) Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error) {
	// 首先尝试从缓存中获取
	found, err := c.store.Get(ctx, key, dst)
	if err != nil {
		return false, err
	}

	// 如果缓存命中，直接返回
	if found {
		return true, nil
	}

	// 缓存未命中，执行回退函数
	if fallback != nil {
		value, fallbackFound, fallbackErr := fallback(ctx, key)
		if fallbackErr != nil {
			return false, fallbackErr
		}

		// 如果回退函数找到了值，缓存它
		if fallbackFound {
			// 使用默认选项如果未提供
			if opts == nil {
				opts = &CacheOptions{}
			}

			// 将值存入缓存
			items := map[string]interface{}{key: value}
			if err := c.store.MSet(ctx, items, opts.TTL); err != nil {
				return false, err
			}

			// 将值赋给目标变量
			if err := assignValue(dst, value); err != nil {
				return false, err
			}

			return true, nil
		}
	}

	return false, nil
}

// MGet 批量获取缓存项，支持部分命中和批量回退
func (c *cacherImpl) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	// 首先尝试从缓存中批量获取
	if err := c.store.MGet(ctx, keys, dstMap); err != nil {
		return err
	}

	// 检查哪些键未命中
	missedKeys := c.getMissedKeys(keys, dstMap)

	// 如果有未命中的键且提供了回退函数，执行批量回退
	if len(missedKeys) > 0 && fallback != nil {
		fallbackValues, err := fallback(ctx, missedKeys)
		if err != nil {
			return err
		}

		// 使用默认选项如果未提供
		if opts == nil {
			opts = &CacheOptions{}
		}

		// 将回退值存入缓存
		if len(fallbackValues) > 0 {
			if err := c.store.MSet(ctx, fallbackValues, opts.TTL); err != nil {
				return err
			}

			// 将回退值合并到结果中
			if err := mergeValues(dstMap, fallbackValues); err != nil {
				return err
			}
		}
	}

	return nil
}

// MDelete 批量清除缓存项
func (c *cacherImpl) MDelete(ctx context.Context, keys []string) (int64, error) {
	return c.store.Del(ctx, keys...)
}

// MRefresh 批量强制刷新缓存项
func (c *cacherImpl) MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	// 删除现有缓存项
	_, err := c.store.Del(ctx, keys...)
	if err != nil {
		return err
	}

	// 使用回退函数获取新值
	if fallback != nil {
		values, err := fallback(ctx, keys)
		if err != nil {
			return err
		}

		// 使用默认选项如果未提供
		if opts == nil {
			opts = &CacheOptions{}
		}

		// 将新值存入缓存
		if len(values) > 0 {
			if err := c.store.MSet(ctx, values, opts.TTL); err != nil {
				return err
			}

			// 将值赋给目标map
			if err := assignValue(dstMap, values); err != nil {
				return err
			}
		}
	}

	return nil
}

// getMissedKeys 获取未命中的键
func (c *cacherImpl) getMissedKeys(keys []string, dstMap interface{}) []string {
	// 使用反射获取dstMap中的键
	dstValue := reflect.ValueOf(dstMap)
	if dstValue.Kind() != reflect.Ptr || dstValue.IsNil() {
		return keys
	}

	dstValue = dstValue.Elem()
	if dstValue.Kind() != reflect.Map {
		return keys
	}

	// 获取已命中的键
	hitKeys := make(map[string]bool)
	for _, key := range dstValue.MapKeys() {
		if key.Kind() == reflect.String {
			hitKeys[key.String()] = true
		}
	}

	// 找出未命中的键
	var missed []string
	for _, key := range keys {
		if !hitKeys[key] {
			missed = append(missed, key)
		}
	}

	return missed
}

// assignValue 将值赋给目标变量
func assignValue(dst interface{}, src interface{}) error {
	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() != reflect.Ptr || dstValue.IsNil() {
		return nil
	}

	srcValue := reflect.ValueOf(src)
	dstValue = dstValue.Elem()

	// 如果类型匹配，直接赋值
	if dstValue.Type() == srcValue.Type() {
		dstValue.Set(srcValue)
		return nil
	}

	// 尝试类型转换
	if srcValue.Type().ConvertibleTo(dstValue.Type()) {
		dstValue.Set(srcValue.Convert(dstValue.Type()))
		return nil
	}

	return nil
}

// mergeValues 将回退值合并到结果中
func mergeValues(dstMap interface{}, fallbackValues map[string]interface{}) error {
	dstValue := reflect.ValueOf(dstMap)
	if dstValue.Kind() != reflect.Ptr || dstValue.IsNil() {
		return nil
	}

	dstValue = dstValue.Elem()
	if dstValue.Kind() != reflect.Map {
		return nil
	}

	// 获取map的键和值类型
	mapType := dstValue.Type()
	keyType := mapType.Key()
	valueType := mapType.Elem()

	// 为每个回退值创建新的map元素
	for key, value := range fallbackValues {
		// 创建键
		keyValue := reflect.ValueOf(key)
		if !keyValue.Type().ConvertibleTo(keyType) {
			continue
		}
		keyValue = keyValue.Convert(keyType)

		// 创建值
		valueValue := reflect.ValueOf(value)
		if !valueValue.Type().ConvertibleTo(valueType) {
			continue
		}
		valueValue = valueValue.Convert(valueType)

		// 设置map元素
		dstValue.SetMapIndex(keyValue, valueValue)
	}

	return nil
}
