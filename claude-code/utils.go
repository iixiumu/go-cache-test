package cache

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// SerializeValue 序列化值为JSON字符串
func SerializeValue(value interface{}) (string, error) {
	if value == nil {
		return "", nil
	}
	
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("failed to serialize value: %w", err)
	}
	
	return string(data), nil
}

// DeserializeValue 反序列化JSON字符串到目标变量
func DeserializeValue(data string, dst interface{}) error {
	if data == "" {
		return nil
	}
	
	if err := json.Unmarshal([]byte(data), dst); err != nil {
		return fmt.Errorf("failed to deserialize value: %w", err)
	}
	
	return nil
}

// ValidateMapPointer 验证目标是否为map指针类型
func ValidateMapPointer(dstMap interface{}) error {
	v := reflect.ValueOf(dstMap)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("dstMap must be a pointer to map")
	}
	
	elem := v.Elem()
	if elem.Kind() != reflect.Map {
		return fmt.Errorf("dstMap must be a pointer to map")
	}
	
	if elem.Type().Key().Kind() != reflect.String {
		return fmt.Errorf("map key must be string")
	}
	
	return nil
}

// SetMapValue 设置map中的值，使用反射处理泛型
func SetMapValue(dstMap interface{}, key string, value interface{}) error {
	mapValue := reflect.ValueOf(dstMap).Elem()
	
	if mapValue.IsNil() {
		mapValue.Set(reflect.MakeMap(mapValue.Type()))
	}
	
	valueType := mapValue.Type().Elem()
	
	// 如果值是字符串，尝试反序列化
	if strVal, ok := value.(string); ok {
		if valueType.Kind() == reflect.String {
			// 如果目标类型是string，直接反序列化
			var result string
			if err := DeserializeValue(strVal, &result); err != nil {
				return err
			}
			mapValue.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(result))
		} else {
			// 如果目标类型不是string，反序列化到目标类型
			newVal := reflect.New(valueType)
			if err := DeserializeValue(strVal, newVal.Interface()); err != nil {
				return err
			}
			mapValue.SetMapIndex(reflect.ValueOf(key), newVal.Elem())
		}
	} else {
		// 直接设置值
		val := reflect.ValueOf(value)
		if val.Type().ConvertibleTo(valueType) {
			mapValue.SetMapIndex(reflect.ValueOf(key), val.Convert(valueType))
		} else {
			return fmt.Errorf("cannot convert %T to %s", value, valueType)
		}
	}
	
	return nil
}