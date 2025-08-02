package cache

import (
	"encoding/json"
	"errors"
	"reflect"
)

var (
	ErrInvalidDstType = errors.New("invalid destination type")
	ErrInvalidMapType = errors.New("invalid map type")
)

// serializeValue 将任意类型的值序列化为字节切片
func serializeValue(value interface{}) ([]byte, error) {
	if value == nil {
		return nil, nil
	}
	
	// 如果是字节切片，直接返回
	if b, ok := value.([]byte); ok {
		return b, nil
	}
	
	// 其他类型序列化为JSON
	return json.Marshal(value)
}

// deserializeValue 将字节切片反序列化为指定类型的值
func deserializeValue(data []byte, dst interface{}) error {
	if dst == nil {
		return ErrInvalidDstType
	}
	
	if len(data) == 0 {
		return nil
	}
	
	// 获取目标类型的反射值
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Ptr {
		return ErrInvalidDstType
	}
	
	// 如果目标是字节切片指针
	if v.Elem().Kind() == reflect.Slice && v.Elem().Type().Elem().Kind() == reflect.Uint8 {
		v.Elem().SetBytes(data)
		return nil
	}
	
	// 反序列化JSON
	return json.Unmarshal(data, dst)
}

// deserializeMap 将map[string][]byte反序列化为map[string]T
func deserializeMap(data map[string][]byte, dstMap interface{}) error {
	if dstMap == nil {
		return ErrInvalidMapType
	}
	
	v := reflect.ValueOf(dstMap)
	if v.Kind() != reflect.Ptr {
		return ErrInvalidMapType
	}
	
	mapValue := v.Elem()
	if mapValue.Kind() != reflect.Map {
		return ErrInvalidMapType
	}
	
	// 确保map的key类型是string
	mapType := mapValue.Type()
	if mapType.Key().Kind() != reflect.String {
		return ErrInvalidMapType
	}
	
	// 创建新的map
	newMap := reflect.MakeMap(mapType)
	
	// 反序列化每个值
	for key, value := range data {
		elemType := mapType.Elem()
		elemPtr := reflect.New(elemType)
		
		if err := deserializeValue(value, elemPtr.Interface()); err != nil {
			return err
		}
		
		newMap.SetMapIndex(reflect.ValueOf(key), elemPtr.Elem())
	}
	
	// 设置结果
	mapValue.Set(newMap)
	return nil
}

// serializeMap 将map[string]interface{}序列化为map[string][]byte
func serializeMap(items map[string]interface{}) (map[string][]byte, error) {
	result := make(map[string][]byte, len(items))
	
	for key, value := range items {
		data, err := serializeValue(value)
		if err != nil {
			return nil, err
		}
		result[key] = data
	}
	
	return result, nil
}