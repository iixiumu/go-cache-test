package cacher

import (
	"context"
	"errors"
	"go-cache/cacher/store"
	"reflect"
	"time"
)

// FallbackFunc 回退函数类型
// 当缓存未命中时执行，用于从数据源获取数据
// key: 请求的键
// 返回: 获取到的值, 是否找到, 错误信息
type FallbackFunc func(ctx context.Context, key string) (interface{}, bool, error)

// BatchFallbackFunc 批量回退函数类型
// 当批量缓存部分未命中时执行，用于从数据源批量获取数据
// keys: 未命中的键列表
// 返回: 键值映射, 错误信息
type BatchFallbackFunc func(ctx context.Context, keys []string) (map[string]interface{}, error)

// CacheOptions 缓存选项
type CacheOptions struct {
	// TTL 缓存过期时间，0表示永不过期
	TTL time.Duration
}

// Cacher 高级缓存接口，提供带回退机制的缓存操作
type Cacher interface {
	// Get 获取单个缓存项，缓存未命中时执行回退函数并缓存结果
	// key: 键名
	// dst: 目标变量的指针，用于接收值
	// fallback: 缓存未命中时的回退函数
	// opts: 缓存选项，可以为nil使用默认选项
	// 返回: 是否找到值（包括从回退函数获取）, 错误信息
	Get(ctx context.Context, key string, dst interface{}, fallback FallbackFunc, opts *CacheOptions) (bool, error)

	// MGet 批量获取缓存项，支持部分命中和批量回退
	// keys: 要获取的键列表
	// dstMap: 目标map的指针，用于接收结果，类型为*map[string]T
	// fallback: 批量回退函数，处理未命中的键
	// opts: 缓存选项，可以为nil使用默认选项
	// 返回: 错误信息
	MGet(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error

	// MDelete 批量清除缓存项
	// keys: 要删除的键列表
	// 返回: 实际删除的键数量, 错误信息
	MDelete(ctx context.Context, keys []string) (int64, error)

	// MRefresh 批量强制刷新缓存项
	// keys: 要刷新的键列表
	// dstMap: 目标map的指针，用于接收结果，类型为*map[string]T
	// fallback: 批量回退函数
	// opts: 缓存选项，可以为nil使用默认选项
	// 返回: 错误信息
	MRefresh(ctx context.Context, keys []string, dstMap interface{}, fallback BatchFallbackFunc, opts *CacheOptions) error

	// GetStore 获取底层Store实例（用于高级操作）
	GetStore() store.Store
}

// CacheError 缓存错误类型
type CacheError struct {
	Op  string // 操作名称
	Key string // 相关键名
	Err error  // 原始错误
}

func (e *CacheError) Error() string {
	if e.Key != "" {
		return "cache " + e.Op + " [" + e.Key + "]: " + e.Err.Error()
	}
	return "cache " + e.Op + ": " + e.Err.Error()
}

func (e *CacheError) Unwrap() error {
	return e.Err
}

// 常用的错误变量
var (
	ErrKeyNotFound    = &CacheError{Op: "get", Err: errors.New("key not found")}
	ErrInvalidType    = &CacheError{Op: "type", Err: errors.New("invalid type")}
	ErrNilDestination = &CacheError{Op: "get", Err: errors.New("destination is nil")}
	ErrNilFallback    = &CacheError{Op: "fallback", Err: errors.New("fallback function is nil")}
)

// 辅助函数：获取值的类型
func GetValueType(v interface{}) reflect.Type {
	if v == nil {
		return nil
	}

	t := reflect.TypeOf(v)
	// 如果是指针，获取指向的类型
	if t.Kind() == reflect.Ptr {
		return t.Elem()
	}
	return t
}

// 辅助函数：验证目标变量是否为有效指针
func ValidateDestination(dst interface{}) error {
	if dst == nil {
		return ErrNilDestination
	}

	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Ptr {
		return &CacheError{Op: "validate", Err: errors.New("destination must be a pointer")}
	}

	if v.IsNil() {
		return ErrNilDestination
	}

	return nil
}

// 辅助函数：验证和解析目标map参数
// dstMap 必须是 *map[string]T 类型的指针
func ValidateDestinationMap(dstMap interface{}) (reflect.Value, reflect.Type, error) {
	if dstMap == nil {
		return reflect.Value{}, nil, ErrNilDestination
	}

	v := reflect.ValueOf(dstMap)
	if v.Kind() != reflect.Ptr {
		return reflect.Value{}, nil, &CacheError{Op: "validate", Err: errors.New("dstMap must be a pointer to map")}
	}

	if v.IsNil() {
		return reflect.Value{}, nil, ErrNilDestination
	}

	elem := v.Elem()
	if elem.Kind() != reflect.Map {
		return reflect.Value{}, nil, &CacheError{Op: "validate", Err: errors.New("dstMap must point to a map")}
	}

	mapType := elem.Type()
	if mapType.Key().Kind() != reflect.String {
		return reflect.Value{}, nil, &CacheError{Op: "validate", Err: errors.New("map key type must be string")}
	}

	// 返回map的Value，以及Value的类型
	return elem, mapType.Elem(), nil
}
