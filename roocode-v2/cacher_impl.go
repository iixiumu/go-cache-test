package cacher

import (
	"context"
	"reflect"
	"time"

	"go-cache/store"
)

// cacherImpl 是Cacher接口的实现
type cacherImpl struct {
	store store.Store
}

// NewCacher 创建一个新的Cacher实例
func NewCacher(store store.Store) Cacher {
	return &cacherImpl{
		store: store,
	}
}

// Get 获取单个缓存项，缓存未命中时执行回退函数并缓存结果
func (c *cacherImpl) Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error) {
	// 尝试从存储后端获取
	found, err := c.store.Get(ctx, key, dst)
	if err != nil {
		return false, err
	}

	// 如果找到了，直接返回
	if found {
		return true, nil
	}

	// 如果没有找到，执行回退函数
	if fallback != nil {
		fallbackValue, fallbackFound, fallbackErr := fallback(ctx, key)
		if fallbackErr != nil {
			return false, fallbackErr
		}

		// 如果回退函数找到了值，缓存它
		if fallbackFound {
			ttl := time.Duration(0)
			if opts != nil {
				ttl = opts.TTL
			}

			items := map[string]interface{}{key: fallbackValue}
			if setErr := c.store.MSet(ctx, items, ttl); setErr != nil {
				// 即使缓存失败，也返回回退的值
				// 使用反射将回退值设置到dst
				dstValue := reflect.ValueOf(dst)
				if dstValue.Kind() == reflect.Ptr {
					dstValue = dstValue.Elem()
					fallbackValueReflect := reflect.ValueOf(fallbackValue)
					if dstValue.Type() == fallbackValueReflect.Type() {
						dstValue.Set(fallbackValueReflect)
					}
				}
				return true, nil
			}

			// 使用反射将回退值设置到dst
			dstValue := reflect.ValueOf(dst)
			if dstValue.Kind() == reflect.Ptr {
				dstValue = dstValue.Elem()
				fallbackValueReflect := reflect.ValueOf(fallbackValue)
				if dstValue.Type() == fallbackValueReflect.Type() {
					dstValue.Set(fallbackValueReflect)
				}
			}
			return true, nil
		}
	}

	return false, nil
}

// MGet 批量获取缓存项，支持部分命中和批量回退
func (c *cacherImpl) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	// 先尝试从存储后端批量获取
	if err := c.store.MGet(ctx, keys, dstMap); err != nil {
		return err
	}

	// 检查哪些键未命中
	exists, err := c.store.Exists(ctx, keys)
	if err != nil {
		return err
	}

	// 找出未命中的键
	var missingKeys []string
	for _, key := range keys {
		if !exists[key] {
			missingKeys = append(missingKeys, key)
		}
	}

	// 如果有未命中的键且提供了回退函数，执行回退
	if len(missingKeys) > 0 && fallback != nil {
		fallbackValues, fallbackErr := fallback(ctx, missingKeys)
		if fallbackErr != nil {
			return fallbackErr
		}

		// 将回退的值缓存
		if len(fallbackValues) > 0 {
			ttl := time.Duration(0)
			if opts != nil {
				ttl = opts.TTL
			}

			if err := c.store.MSet(ctx, fallbackValues, ttl); err != nil {
				// 即使缓存失败，也继续执行
			}

			// 将回退的值合并到结果中
			dstMapValue := reflect.ValueOf(dstMap)
			if dstMapValue.Kind() == reflect.Ptr {
				dstMapValue = dstMapValue.Elem()
				if dstMapValue.Kind() == reflect.Map {
					for key, value := range fallbackValues {
						keyValue := reflect.ValueOf(key).Convert(dstMapValue.Type().Key())
						valueValue := reflect.ValueOf(value).Convert(dstMapValue.Type().Elem())
						dstMapValue.SetMapIndex(keyValue, valueValue)
					}
				}
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
	// 直接执行回退函数获取最新值
	if fallback == nil {
		return nil
	}

	values, err := fallback(ctx, keys)
	if err != nil {
		return err
	}

	// 缓存新值
	if len(values) > 0 {
		ttl := time.Duration(0)
		if opts != nil {
			ttl = opts.TTL
		}

		if err := c.store.MSet(ctx, values, ttl); err != nil {
			return err
		}

		// 将值设置到dstMap
		dstMapValue := reflect.ValueOf(dstMap)
		if dstMapValue.Kind() == reflect.Ptr {
			dstMapValue = dstMapValue.Elem()
			if dstMapValue.Kind() == reflect.Map {
				for key, value := range values {
					keyValue := reflect.ValueOf(key).Convert(dstMapValue.Type().Key())
					valueValue := reflect.ValueOf(value).Convert(dstMapValue.Type().Elem())
					dstMapValue.SetMapIndex(keyValue, valueValue)
				}
			}
		}
	}

	return nil
}
