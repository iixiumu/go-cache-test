package gcache

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/bluele/gcache"
	"go-cache/store"
)

// gcacheStore gcache存储
// 实现了store.Store接口
type gcacheStore struct {
	client gcache.Cache
}

// NewGcacheStore 创建gcache存储
func NewGcacheStore(client gcache.Cache) store.Store {
	return &gcacheStore{client: client}
}

// Get 从gcache获取单个值
func (s *gcacheStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	val, err := s.client.Get(key)
	if err == gcache.KeyNotFoundError {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, s.unmarshal(val, dst)
}

// MGet 批量获取值到map中
func (s *gcacheStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	if len(keys) == 0 {
		return nil
	}

	vals := make(map[string]interface{})
	for _, key := range keys {
		if val, err := s.client.Get(key); err == nil {
			vals[key] = val
		}
	}

	return s.unmarshalMap(vals, dstMap)
}

// Exists 批量检查键存在性
func (s *gcacheStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	if len(keys) == 0 {
		return make(map[string]bool), nil
	}

	existsMap := make(map[string]bool)
	for _, key := range keys {
		existsMap[key] = s.client.Has(key)
	}

	return existsMap, nil
}

// MSet 批量设置键值对，支持TTL
func (s *gcacheStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	for key, value := range items {
		if ttl > 0 {
			s.client.SetWithExpire(key, value, ttl)
		} else {
			s.client.Set(key, value)
		}
	}

	return nil
}

// Del 删除指定键
func (s *gcacheStore) Del(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	var deletedCount int64
	for _, key := range keys {
		if s.client.Remove(key) {
			deletedCount++
		}
	}

	return deletedCount, nil
}

// unmarshal 反序列化
func (s *gcacheStore) unmarshal(src interface{}, dst interface{}) error {
	dstVal := reflect.ValueOf(dst)
	if dstVal.Kind() != reflect.Ptr {
		return errors.New("dst must be a pointer")
	}

	srcVal := reflect.ValueOf(src)
	dstElem := dstVal.Elem()

	if !srcVal.Type().AssignableTo(dstElem.Type()) {
		return fmt.Errorf("cannot assign %s to %s", srcVal.Type(), dstElem.Type())
	}

	dstElem.Set(srcVal)
	return nil
}

// unmarshalMap 批量反序列化到map
func (s *gcacheStore) unmarshalMap(srcMap map[string]interface{}, dstMap interface{}) error {
	dstVal := reflect.ValueOf(dstMap)
	if dstVal.Kind() != reflect.Ptr || dstVal.Elem().Kind() != reflect.Map {
		return errors.New("dstMap must be a pointer to a map")
	}

	mapVal := dstVal.Elem()
	mapType := mapVal.Type()
	keyType := mapType.Key()
	valType := mapType.Elem()

	if keyType.Kind() != reflect.String {
		return fmt.Errorf("map key type must be string, but got %s", keyType.Kind())
	}

	for key, val := range srcMap {
		srcVal := reflect.ValueOf(val)
		if !srcVal.Type().AssignableTo(valType) {
			return fmt.Errorf("cannot assign %s to %s", srcVal.Type(), valType)
		}
		mapVal.SetMapIndex(reflect.ValueOf(key), srcVal)
	}

	return nil
}