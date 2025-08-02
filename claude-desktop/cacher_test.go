package cacher

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"
)

// MockStore 模拟Store实现，用于测试
type MockStore struct {
	data      map[string]interface{}
	getErr    error
	mgetErr   error
	existsErr error
	msetErr   error
	delErr    error
	delCount  int64
}

func NewMockStore() *MockStore {
	return &MockStore{
		data: make(map[string]interface{}),
	}
}

func (m *MockStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	if m.getErr != nil {
		return false, m.getErr
	}

	value, exists := m.data[key]
	if !exists {
		return false, nil
	}

	return true, assignValue(dst, value)
}

func (m *MockStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	if m.mgetErr != nil {
		return m.mgetErr
	}

	mapValue, _, err := ValidateDestinationMap(dstMap)
	if err != nil {
		return err
	}

	if mapValue.IsNil() {
		mapValue.Set(reflect.MakeMap(mapValue.Type()))
	}

	for _, key := range keys {
		if value, exists := m.data[key]; exists {
			mapValue.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
		}
	}

	return nil
}

func (m *MockStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	if m.existsErr != nil {
		return nil, m.existsErr
	}

	result := make(map[string]bool)
	for _, key := range keys {
		_, exists := m.data[key]
		result[key] = exists
	}

	return result, nil
}

func (m *MockStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if m.msetErr != nil {
		return m.msetErr
	}

	for key, value := range items {
		m.data[key] = value
	}

	return nil
}

func (m *MockStore) Del(ctx context.Context, keys ...string) (int64, error) {
	if m.delErr != nil {
		return 0, m.delErr
	}

	if m.delCount > 0 {
		return m.delCount, nil
	}

	count := int64(0)
	for _, key := range keys {
		if _, exists := m.data[key]; exists {
			delete(m.data, key)
			count++
		}
	}

	return count, nil
}

// 测试用的数据结构
type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestDefaultCacher_Get(t *testing.T) {
	ctx := context.Background()
	store := NewMockStore()
	cacher := NewCacher(store)

	t.Run("缓存命中", func(t *testing.T) {
		// 预设缓存数据
		expectedUser := &User{ID: 1, Name: "Alice", Age: 25}
		store.data["user:1"] = expectedUser

		var result User
		found, err := cacher.Get(ctx, "user:1", &result, nil, nil)

		if err != nil {
			t.Errorf("Get() error = %v", err)
		}
		if !found {
			t.Error("Get() found = false, want true")
		}
		if result.ID != expectedUser.ID || result.Name != expectedUser.Name {
			t.Errorf("Get() result = %+v, want %+v", result, expectedUser)
		}
	})

	t.Run("缓存未命中_无回退函数", func(t *testing.T) {
		var result User
		found, err := cacher.Get(ctx, "user:notexist", &result, nil, nil)

		if err != nil {
			t.Errorf("Get() error = %v", err)
		}
		if found {
			t.Error("Get() found = true, want false")
		}
	})

	t.Run("缓存未命中_有回退函数", func(t *testing.T) {
		expectedUser := &User{ID: 2, Name: "Bob", Age: 30}
		fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
			if key == "user:2" {
				return expectedUser, true, nil
			}
			return nil, false, nil
		}

		var result User
		found, err := cacher.Get(ctx, "user:2", &result, fallback, &CacheOptions{TTL: 5 * time.Minute})

		if err != nil {
			t.Errorf("Get() error = %v", err)
		}
		if !found {
			t.Error("Get() found = false, want true")
		}
		if result.ID != expectedUser.ID || result.Name != expectedUser.Name {
			t.Errorf("Get() result = %+v, want %+v", result, expectedUser)
		}

		// 验证数据已缓存
		if cachedValue, exists := store.data["user:2"]; !exists {
			t.Error("数据未写入缓存")
		} else if cachedUser := cachedValue.(*User); cachedUser.ID != expectedUser.ID {
			t.Error("缓存的数据不正确")
		}
	})

	t.Run("回退函数返回未找到", func(t *testing.T) {
		fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
			return nil, false, nil
		}

		var result User
		found, err := cacher.Get(ctx, "user:notfound", &result, fallback, nil)

		if err != nil {
			t.Errorf("Get() error = %v", err)
		}
		if found {
			t.Error("Get() found = true, want false")
		}
	})

	t.Run("回退函数返回错误", func(t *testing.T) {
		expectedErr := errors.New("fallback error")
		fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
			return nil, false, expectedErr
		}

		var result User
		found, err := cacher.Get(ctx, "user:error", &result, fallback, nil)

		if err == nil {
			t.Error("Get() error = nil, want error")
		}
		if found {
			t.Error("Get() found = true, want false")
		}
	})

	t.Run("目标变量验证错误", func(t *testing.T) {
		// 传入nil
		found, err := cacher.Get(ctx, "user:1", nil, nil, nil)
		if err == nil {
			t.Error("Get() error = nil, want error")
		}
		if found {
			t.Error("Get() found = true, want false")
		}

		// 传入非指针
		var result User
		found, err = cacher.Get(ctx, "user:1", result, nil, nil)
		if err == nil {
			t.Error("Get() error = nil, want error")
		}
		if found {
			t.Error("Get() found = true, want false")
		}
	})
}

func TestDefaultCacher_MGet(t *testing.T) {
	ctx := context.Background()
	store := NewMockStore()
	cacher := NewCacher(store)

	t.Run("全部缓存命中", func(t *testing.T) {
		// 预设缓存数据
		user1 := &User{ID: 1, Name: "Alice", Age: 25}
		user2 := &User{ID: 2, Name: "Bob", Age: 30}
		store.data["user:1"] = user1
		store.data["user:2"] = user2

		keys := []string{"user:1", "user:2"}
		var result map[string]*User
		err := cacher.MGet(ctx, keys, &result, nil, nil)

		if err != nil {
			t.Errorf("MGet() error = %v", err)
		}
		if len(result) != 2 {
			t.Errorf("MGet() result length = %d, want 2", len(result))
		}
		if result["user:1"].Name != "Alice" || result["user:2"].Name != "Bob" {
			t.Errorf("MGet() result data incorrect")
		}
	})

	t.Run("部分缓存命中_有回退函数", func(t *testing.T) {
		// 清空之前的数据
		store.data = make(map[string]interface{})

		// 只预设一个用户
		user1 := &User{ID: 1, Name: "Alice", Age: 25}
		store.data["user:1"] = user1

		fallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
			result := make(map[string]interface{})
			for _, key := range keys {
				if key == "user:2" {
					result[key] = &User{ID: 2, Name: "Bob", Age: 30}
				}
				if key == "user:3" {
					result[key] = &User{ID: 3, Name: "Charlie", Age: 35}
				}
			}
			return result, nil
		}

		keys := []string{"user:1", "user:2", "user:3"}
		var result map[string]*User
		err := cacher.MGet(ctx, keys, &result, fallback, &CacheOptions{TTL: 5 * time.Minute})

		if err != nil {
			t.Errorf("MGet() error = %v", err)
		}
		if len(result) != 3 {
			t.Errorf("MGet() result length = %d, want 3", len(result))
		}

		// 验证数据正确性
		if result["user:1"].Name != "Alice" {
			t.Error("缓存命中的数据不正确")
		}
		if result["user:2"].Name != "Bob" {
			t.Error("回退函数获取的数据不正确")
		}
		if result["user:3"].Name != "Charlie" {
			t.Error("回退函数获取的数据不正确")
		}

		// 验证回退数据已缓存
		if _, exists := store.data["user:2"]; !exists {
			t.Error("回退数据未写入缓存")
		}
		if _, exists := store.data["user:3"]; !exists {
			t.Error("回退数据未写入缓存")
		}
	})

	t.Run("空键列表", func(t *testing.T) {
		var result map[string]*User
		err := cacher.MGet(ctx, []string{}, &result, nil, nil)

		if err != nil {
			t.Errorf("MGet() error = %v", err)
		}
	})

	t.Run("目标map验证错误", func(t *testing.T) {
		keys := []string{"user:1"}

		// 传入nil
		err := cacher.MGet(ctx, keys, nil, nil, nil)
		if err == nil {
			t.Error("MGet() error = nil, want error")
		}

		// 传入非指针
		var result map[string]*User
		err = cacher.MGet(ctx, keys, result, nil, nil)
		if err == nil {
			t.Error("MGet() error = nil, want error")
		}

		// 传入非map指针
		var notMap string
		err = cacher.MGet(ctx, keys, &notMap, nil, nil)
		if err == nil {
			t.Error("MGet() error = nil, want error")
		}
	})

	t.Run("回退函数返回错误", func(t *testing.T) {
		store.data = make(map[string]interface{})
		expectedErr := errors.New("batch fallback error")

		fallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
			return nil, expectedErr
		}

		keys := []string{"user:1"}
		var result map[string]*User
		err := cacher.MGet(ctx, keys, &result, fallback, nil)

		if err == nil {
			t.Error("MGet() error = nil, want error")
		}
	})
}

func TestDefaultCacher_MDelete(t *testing.T) {
	ctx := context.Background()
	store := NewMockStore()
	cacher := NewCacher(store)

	t.Run("删除存在的键", func(t *testing.T) {
		// 预设数据
		store.data["user:1"] = &User{ID: 1, Name: "Alice"}
		store.data["user:2"] = &User{ID: 2, Name: "Bob"}

		keys := []string{"user:1", "user:2"}
		count, err := cacher.MDelete(ctx, keys)

		if err != nil {
			t.Errorf("MDelete() error = %v", err)
		}
		if count != 2 {
			t.Errorf("MDelete() count = %d, want 2", count)
		}

		// 验证数据已删除
		if _, exists := store.data["user:1"]; exists {
			t.Error("user:1未被删除")
		}
		if _, exists := store.data["user:2"]; exists {
			t.Error("user:2未被删除")
		}
	})

	t.Run("删除不存在的键", func(t *testing.T) {
		keys := []string{"user:notexist"}
		count, err := cacher.MDelete(ctx, keys)

		if err != nil {
			t.Errorf("MDelete() error = %v", err)
		}
		if count != 0 {
			t.Errorf("MDelete() count = %d, want 0", count)
		}
	})

	t.Run("空键列表", func(t *testing.T) {
		count, err := cacher.MDelete(ctx, []string{})

		if err != nil {
			t.Errorf("MDelete() error = %v", err)
		}
		if count != 0 {
			t.Errorf("MDelete() count = %d, want 0", count)
		}
	})

	t.Run("Store删除错误", func(t *testing.T) {
		store.delErr = errors.New("delete error")
		defer func() { store.delErr = nil }()

		keys := []string{"user:1"}
		count, err := cacher.MDelete(ctx, keys)

		if err == nil {
			t.Error("MDelete() error = nil, want error")
		}
		if count != 0 {
			t.Errorf("MDelete() count = %d, want 0", count)
		}
	})
}

func TestDefaultCacher_MRefresh(t *testing.T) {
	ctx := context.Background()
	store := NewMockStore()
	cacher := NewCacher(store)

	t.Run("刷新成功", func(t *testing.T) {
		// 预设旧数据
		oldUser := &User{ID: 1, Name: "Alice", Age: 25}
		store.data["user:1"] = oldUser

		// 回退函数返回新数据
		newUser := &User{ID: 1, Name: "Alice Updated", Age: 26}
		fallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
			result := make(map[string]interface{})
			for _, key := range keys {
				if key == "user:1" {
					result[key] = newUser
				}
			}
			return result, nil
		}

		keys := []string{"user:1"}
		var result map[string]*User
		err := cacher.MRefresh(ctx, keys, &result, fallback, &CacheOptions{TTL: 5 * time.Minute})

		if err != nil {
			t.Errorf("MRefresh() error = %v", err)
		}
		if len(result) != 1 {
			t.Errorf("MRefresh() result length = %d, want 1", len(result))
		}
		if result["user:1"].Name != "Alice Updated" || result["user:1"].Age != 26 {
			t.Errorf("MRefresh() result = %+v, want updated data", result["user:1"])
		}

		// 验证缓存中的数据已更新
		if cachedUser := store.data["user:1"].(*User); cachedUser.Name != "Alice Updated" {
			t.Error("缓存数据未更新")
		}
	})

	t.Run("空键列表", func(t *testing.T) {
		var result map[string]*User
		err := cacher.MRefresh(ctx, []string{}, &result, nil, nil)

		if err != nil {
			t.Errorf("MRefresh() error = %v", err)
		}
	})
}

func TestDefaultCacher_GetStore(t *testing.T) {
	store := NewMockStore()
	cacher := NewCacher(store)

	if cacher.GetStore() != store {
		t.Error("GetStore() returned different store instance")
	}
}

// 测试辅助函数
func TestAssignValue(t *testing.T) {
	t.Run("成功赋值", func(t *testing.T) {
		var target int
		err := assignValue(&target, 42)

		if err != nil {
			t.Errorf("assignValue() error = %v", err)
		}
		if target != 42 {
			t.Errorf("assignValue() target = %d, want 42", target)
		}
	})

	t.Run("类型不匹配", func(t *testing.T) {
		var target int
		err := assignValue(&target, "string")

		if err == nil {
			t.Error("assignValue() error = nil, want error")
		}
	})

	t.Run("目标不是指针", func(t *testing.T) {
		var target int
		err := assignValue(target, 42)

		if err == nil {
			t.Error("assignValue() error = nil, want error")
		}
	})
}

func TestIsAssignableToType(t *testing.T) {
	t.Run("兼容类型", func(t *testing.T) {
		if !isAssignableToType(42, reflect.TypeOf(0)) {
			t.Error("isAssignableToType() = false, want true for int to int")
		}
	})

	t.Run("不兼容类型", func(t *testing.T) {
		if isAssignableToType("string", reflect.TypeOf(0)) {
			t.Error("isAssignableToType() = true, want false for string to int")
		}
	})

	t.Run("nil值", func(t *testing.T) {
		ptrType := reflect.TypeOf((*int)(nil))
		if !isAssignableToType(nil, ptrType) {
			t.Error("isAssignableToType() = false, want true for nil to pointer")
		}

		intType := reflect.TypeOf(0)
		if isAssignableToType(nil, intType) {
			t.Error("isAssignableToType() = true, want false for nil to int")
		}
	})
}

// 基准测试
func BenchmarkCacher_Get(b *testing.B) {
	ctx := context.Background()
	store := NewMockStore()
	cacher := NewCacher(store)

	// 预设数据
	user := &User{ID: 1, Name: "Alice", Age: 25}
	store.data["user:1"] = user

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result User
		cacher.Get(ctx, "user:1", &result, nil, nil)
	}
}

func BenchmarkCacher_MGet(b *testing.B) {
	ctx := context.Background()
	store := NewMockStore()
	cacher := NewCacher(store)

	// 预设数据
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("user:%d", i)
		user := &User{ID: i, Name: fmt.Sprintf("User%d", i), Age: 20 + i%50}
		store.data[key] = user
	}

	keys := make([]string, 10)
	for i := 0; i < 10; i++ {
		keys[i] = fmt.Sprintf("user:%d", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result map[string]*User
		cacher.MGet(ctx, keys, &result, nil, nil)
	}
}
