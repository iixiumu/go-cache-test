package cache

import (
	"context"
	"encoding/json"
	"reflect"
	"time"
)

// cacherImpl Cacher接口的实现
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
	// 尝试从缓存获取
	found, err := c.store.Get(ctx, key, dst)
	if err != nil {
		return false, err
	}

	if found {
		return true, nil
	}

	// 缓存未命中，执行回退函数
	if fallback != nil {
		value, found, err := fallback(ctx, key)
		if err != nil {
			return false, err
		}

		if found {
			// 将回退函数获取的值设置到目标变量
			if err := setValue(dst, value); err != nil {
				return false, err
			}

			// 缓存结果
			ttl := time.Duration(0)
			if opts != nil {
				ttl = opts.TTL
			}

			if err := c.store.MSet(ctx, map[string]interface{}{key: value}, ttl); err != nil {
				// 缓存失败不影响返回值，只记录错误
				// 这里可以添加日志记录
			}

			return true, nil
		}
	}

	return false, nil
}

// MGet 批量获取缓存项，支持部分命中和批量回退
func (c *cacherImpl) MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error {
	if len(keys) == 0 {
		return nil
	}

	// 验证dstMap类型
	dstValue := reflect.ValueOf(dstMap)
	if dstValue.Kind() != reflect.Ptr || dstValue.Elem().Kind() != reflect.Map {
		return ErrInvalidDstMap
	}

	// 从缓存批量获取
	err := c.store.MGet(ctx, keys, dstMap)
	if err != nil {
		return err
	}

	// 检查哪些键未命中
	dstMapValue := dstValue.Elem()
	missingKeys := make([]string, 0)

	for _, key := range keys {
		keyValue := reflect.ValueOf(key)
		if !dstMapValue.MapIndex(keyValue).IsValid() {
			missingKeys = append(missingKeys, key)
		}
	}

	// 如果有未命中的键且提供了回退函数
	if len(missingKeys) > 0 && fallback != nil {
		// 执行批量回退
		fallbackResults, err := fallback(ctx, missingKeys)
		if err != nil {
			return err
		}

		// 将回退结果添加到目标map
		for key, value := range fallbackResults {
			keyValue := reflect.ValueOf(key)
			dstMapValue.SetMapIndex(keyValue, reflect.ValueOf(value))
		}

		// 缓存回退结果
		if len(fallbackResults) > 0 {
			ttl := time.Duration(0)
			if opts != nil {
				ttl = opts.TTL
			}

			if err := c.store.MSet(ctx, fallbackResults, ttl); err != nil {
				// 缓存失败不影响返回值，只记录错误
				// 这里可以添加日志记录
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
	if len(keys) == 0 {
		return nil
	}

	// 验证dstMap类型
	dstValue := reflect.ValueOf(dstMap)
	if dstValue.Kind() != reflect.Ptr || dstValue.Elem().Kind() != reflect.Map {
		return ErrInvalidDstMap
	}

	// 清空目标map
	dstMapValue := dstValue.Elem()
	dstMapValue.Set(reflect.MakeMap(dstMapValue.Type()))

	// 执行批量回退获取最新数据
	if fallback != nil {
		results, err := fallback(ctx, keys)
		if err != nil {
			return err
		}

		// 将结果设置到目标map
		for key, value := range results {
			keyValue := reflect.ValueOf(key)
			dstMapValue.SetMapIndex(keyValue, reflect.ValueOf(value))
		}

		// 缓存最新结果
		if len(results) > 0 {
			ttl := time.Duration(0)
			if opts != nil {
				ttl = opts.TTL
			}

			if err := c.store.MSet(ctx, results, ttl); err != nil {
				// 缓存失败不影响返回值，只记录错误
				// 这里可以添加日志记录
			}
		}
	}

	return nil
}

// setValue 使用反射将值设置到目标变量
func setValue(dst interface{}, value interface{}) error {
	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() != reflect.Ptr {
		return ErrInvalidDst
	}

	dstElem := dstValue.Elem()
	valueValue := reflect.ValueOf(value)

	// 如果类型匹配，直接设置
	if dstElem.Type() == valueValue.Type() {
		dstElem.Set(valueValue)
		return nil
	}

	// 如果目标是指针类型，需要解引用
	if dstElem.Kind() == reflect.Ptr {
		if dstElem.IsNil() {
			dstElem.Set(reflect.New(dstElem.Type().Elem()))
		}
		dstElem = dstElem.Elem()
	}

	// 尝试类型转换
	if valueValue.Type().ConvertibleTo(dstElem.Type()) {
		dstElem.Set(valueValue.Convert(dstElem.Type()))
		return nil
	}

	// 尝试JSON序列化/反序列化
	if valueBytes, err := json.Marshal(value); err == nil {
		return json.Unmarshal(valueBytes, dst)
	}

	return ErrTypeMismatch
}
