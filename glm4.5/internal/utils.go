package internal

import (
	"encoding/json"
	"errors"
	"reflect"
	"time"
)

// CacheOptions 缓存选项
type CacheOptions struct {
	// TTL 缓存过期时间，0表示永不过期
	TTL time.Duration
}

// SerializeValue 序列化值为字节切片
func SerializeValue(value interface{}) ([]byte, error) {
	return json.Marshal(value)
}

// DeserializeValue 反序列化字节切片到目标变量
func DeserializeValue(data []byte, dst interface{}) error {
	if data == nil {
		return nil
	}
	return json.Unmarshal(data, dst)
}

// GetTypeOfMap 获取map的类型信息
func GetTypeOfMap(dstMap interface{}) (reflect.Type, reflect.Type, error) {
	dstMapValue := reflect.ValueOf(dstMap)
	if dstMapValue.Kind() != reflect.Ptr {
		return nil, nil, errors.New("dstMap must be a pointer to a map")
	}

	dstMapElem := dstMapValue.Elem()
	if dstMapElem.Kind() != reflect.Map {
		return nil, nil, errors.New("dstMap must be a pointer to a map")
	}

	if dstMapElem.IsNil() {
		dstMapElem.Set(reflect.MakeMap(dstMapElem.Type()))
	}

	keyType := dstMapElem.Type().Key()
	valueType := dstMapElem.Type().Elem()

	return keyType, valueType, nil
}

// SetMapValue 向map设置值
func SetMapValue(dstMap interface{}, key string, value interface{}) error {
	dstMapValue := reflect.ValueOf(dstMap).Elem()
	keyType := dstMapValue.Type().Key()
	valueType := dstMapValue.Type().Elem()

	keyValue := reflect.ValueOf(key)
	if keyValue.Type().ConvertibleTo(keyType) {
		keyValue = keyValue.Convert(keyType)
	} else {
		return errors.New("key type mismatch")
	}

	valueReflect := reflect.ValueOf(value)
	if valueReflect.Type().ConvertibleTo(valueType) {
		valueReflect = valueReflect.Convert(valueType)
		dstMapValue.SetMapIndex(keyValue, valueReflect)
		return nil
	}

	return errors.New("value type mismatch")
}

// SetMapValueWithType 向map设置值（带类型检查）
func SetMapValueWithType(dstMap interface{}, key string, value interface{}, keyType, valueType reflect.Type) error {
	dstMapValue := reflect.ValueOf(dstMap).Elem()

	keyValue := reflect.ValueOf(key)
	if keyValue.Type().ConvertibleTo(keyType) {
		keyValue = keyValue.Convert(keyType)
	} else {
		return errors.New("key type mismatch")
	}

	valueReflect := reflect.ValueOf(value)
	if valueReflect.Type().ConvertibleTo(valueType) {
		valueReflect = valueReflect.Convert(valueType)
		dstMapValue.SetMapIndex(keyValue, valueReflect)
		return nil
	}

	return errors.New("value type mismatch")
}

// GetDefaultTTL 获取默认TTL
func GetDefaultTTL(opts *CacheOptions) time.Duration {
	if opts == nil {
		return 0
	}
	return opts.TTL
}