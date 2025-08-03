package cacher

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockStore 模拟Store实现，用于测试
type MockStore struct {
	data map[string]interface{}
	ttls map[string]time.Time
}

func NewMockStore() *MockStore {
	return &MockStore{
		data: make(map[string]interface{}),
		ttls: make(map[string]time.Time),
	}
}

func (m *MockStore) Get(ctx context.Context, key string, dst interface{}) (bool, error) {
	// 检查是否过期
	if expiry, exists := m.ttls[key]; exists && time.Now().After(expiry) {
		delete(m.data, key)
		delete(m.ttls, key)
		return false, nil
	}

	value, exists := m.data[key]
	if !exists {
		return false, nil
	}

	// 简单的值复制，实际实现应该更复杂
	switch v := dst.(type) {
	case *string:
		if str, ok := value.(string); ok {
			*v = str
		}
	case *int:
		if i, ok := value.(int); ok {
			*v = i
		}
	case *bool:
		if b, ok := value.(bool); ok {
			*v = b
		}
	default:
		// 对于复杂类型，这里简化处理
		if ptr, ok := dst.(*interface{}); ok {
			*ptr = value
		}
	}

	return true, nil
}

func (m *MockStore) MGet(ctx context.Context, keys []string, dstMap interface{}) error {
	// 简化实现，实际应该使用反射
	if resultMap, ok := dstMap.(*map[string]string); ok {
		if *resultMap == nil {
			*resultMap = make(map[string]string)
		}
		for _, key := range keys {
			if expiry, exists := m.ttls[key]; exists && time.Now().After(expiry) {
				delete(m.data, key)
				delete(m.ttls, key)
				continue
			}
			if value, exists := m.data[key]; exists {
				if str, ok := value.(string); ok {
					(*resultMap)[key] = str
				}
			}
		}
	}
	return nil
}

func (m *MockStore) Exists(ctx context.Context, keys []string) (map[string]bool, error) {
	result := make(map[string]bool)
	for _, key := range keys {
		// 检查是否过期
		if expiry, exists := m.ttls[key]; exists && time.Now().After(expiry) {
			delete(m.data, key)
			delete(m.ttls, key)
			result[key] = false
			continue
		}
		_, exists := m.data[key]
		result[key] = exists
	}
	return result, nil
}

func (m *MockStore) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	for key, value := range items {
		m.data[key] = value
		if ttl > 0 {
			m.ttls[key] = time.Now().Add(ttl)
		}
	}
	return nil
}

func (m *MockStore) Del(ctx context.Context, keys ...string) (int64, error) {
	var count int64
	for _, key := range keys {
		if _, exists := m.data[key]; exists {
			delete(m.data, key)
			delete(m.ttls, key)
			count++
		}
	}
	return count, nil
}

// CacherTestSuite Cacher接口测试套件
type CacherTestSuite struct {
	Cacher Cacher
	t      *testing.T
}

// NewCacherTestSuite 创建新的Cacher测试套件
func NewCacherTestSuite(t *testing.T, cacher Cacher) *CacherTestSuite {
	return &CacherTestSuite{
		Cacher: cacher,
		t:      t,
	}
}

// RunAllTests 执行所有Cacher接口测试
func (ts *CacherTestSuite) RunAllTests() {
	ts.TestGet()
	ts.TestGetWithFallback()
	ts.TestMGet()
	ts.TestMGetPartialHit()
	ts.TestMDelete()
	ts.TestMRefresh()
	ts.TestCacheOptions()
	ts.TestErrorHandling()
}

// TestGet 测试Get方法
func (ts *CacherTestSuite) TestGet() {
	ctx := context.Background()
	t := ts.t

	// 测试缓存未命中，使用fallback
	fallbackCalled := false
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		fallbackCalled = true
		assert.Equal(t, "test_key", key)
		return "fallback_value", true, nil
	}

	var result string
	found, err := ts.Cacher.Get(ctx, "test_key", &result, fallback, nil)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "fallback_value", result)
	assert.True(t, fallbackCalled)

	// 测试缓存命中，不调用fallback
	fallbackCalled = false
	found, err = ts.Cacher.Get(ctx, "test_key", &result, fallback, nil)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "fallback_value", result)
	assert.False(t, fallbackCalled)
}

// TestGetWithFallback 测试fallback函数的不同情况
func (ts *CacherTestSuite) TestGetWithFallback() {
	ctx := context.Background()
	t := ts.t

	// 测试fallback返回false
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		return nil, false, nil
	}

	var result string
	found, err := ts.Cacher.Get(ctx, "not_found_key", &result, fallback, nil)
	assert.NoError(t, err)
	assert.False(t, found)

	// 测试fallback返回错误
	fallbackError := func(ctx context.Context, key string) (interface{}, bool, error) {
		return nil, false, errors.New("fallback error")
	}

	found, err = ts.Cacher.Get(ctx, "error_key", &result, fallbackError, nil)
	assert.Error(t, err)
	assert.False(t, found)
	assert.Contains(t, err.Error(), "fallback error")
}

// TestMGet 测试MGet方法
func (ts *CacherTestSuite) TestMGet() {
	ctx := context.Background()
	t := ts.t

	// 首先设置一些缓存数据
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		return "value_" + key, true, nil
	}

	var result string
	_, err := ts.Cacher.Get(ctx, "key1", &result, fallback, nil)
	require.NoError(t, err)

	// 测试批量获取
	batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		result := make(map[string]interface{})
		for _, key := range keys {
			result[key] = "batch_" + key
		}
		return result, nil
	}

	keys := []string{"key1", "key2", "key3"}
	resultMap := make(map[string]string)
	err = ts.Cacher.MGet(ctx, keys, &resultMap, batchFallback, nil)
	assert.NoError(t, err)

	// 验证结果
	assert.Contains(t, resultMap, "key1")
	assert.Contains(t, resultMap, "key2")
	assert.Contains(t, resultMap, "key3")
}

// TestMGetPartialHit 测试部分缓存命中的情况
func (ts *CacherTestSuite) TestMGetPartialHit() {
	ctx := context.Background()
	t := ts.t

	// 先设置一个缓存
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		return "cached_" + key, true, nil
	}

	var result string
	_, err := ts.Cacher.Get(ctx, "cached_key", &result, fallback, nil)
	require.NoError(t, err)

	// 批量获取，部分命中
	batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		result := make(map[string]interface{})
		for _, key := range keys {
			result[key] = "new_" + key
		}
		return result, nil
	}

	keys := []string{"cached_key", "new_key1", "new_key2"}
	resultMap := make(map[string]string)
	err = ts.Cacher.MGet(ctx, keys, &resultMap, batchFallback, nil)
	assert.NoError(t, err)

	// cached_key应该返回缓存的值，其他应该返回fallback的值
	assert.Len(t, resultMap, 3)
	assert.Contains(t, resultMap, "cached_key")
	assert.Contains(t, resultMap, "new_key1")
	assert.Contains(t, resultMap, "new_key2")
}

// TestMDelete 测试MDelete方法
func (ts *CacherTestSuite) TestMDelete() {
	ctx := context.Background()
	t := ts.t

	// 先设置一些缓存
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		return "value_" + key, true, nil
	}

	var result string
	for _, key := range []string{"del1", "del2", "del3"} {
		_, err := ts.Cacher.Get(ctx, key, &result, fallback, nil)
		require.NoError(t, err)
	}

	// 删除缓存
	deletedCount, err := ts.Cacher.MDelete(ctx, []string{"del1", "del2", "nonexistent"})
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, deletedCount, int64(2)) // 至少删除了2个

	// 验证缓存已被删除
	fallbackCalled := false
	fallbackCheck := func(ctx context.Context, key string) (interface{}, bool, error) {
		fallbackCalled = true
		return "new_value", true, nil
	}

	found, err := ts.Cacher.Get(ctx, "del1", &result, fallbackCheck, nil)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.True(t, fallbackCalled) // 应该调用fallback，说明缓存被删除了
}

// TestMRefresh 测试MRefresh方法
func (ts *CacherTestSuite) TestMRefresh() {
	ctx := context.Background()
	t := ts.t

	// 先设置一些缓存
	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		return "old_value_" + key, true, nil
	}

	var result string
	for _, key := range []string{"refresh1", "refresh2"} {
		_, err := ts.Cacher.Get(ctx, key, &result, fallback, nil)
		require.NoError(t, err)
	}

	// 刷新缓存
	batchFallback := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		result := make(map[string]interface{})
		for _, key := range keys {
			result[key] = "new_value_" + key
		}
		return result, nil
	}

	keys := []string{"refresh1", "refresh2"}
	resultMap := make(map[string]string)
	err := ts.Cacher.MRefresh(ctx, keys, &resultMap, batchFallback, nil)
	assert.NoError(t, err)

	// 验证返回了新值
	assert.Len(t, resultMap, 2)
	for _, key := range keys {
		assert.Contains(t, resultMap, key)
		assert.Equal(t, "new_value_"+key, resultMap[key])
	}

	// 验证缓存也被更新了
	found, err := ts.Cacher.Get(ctx, "refresh1", &result, nil, nil)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "new_value_refresh1", result)
}

// TestCacheOptions 测试缓存选项
func (ts *CacherTestSuite) TestCacheOptions() {
	ctx := context.Background()
	t := ts.t

	// 测试TTL选项
	opts := &CacheOptions{
		TTL: 100 * time.Millisecond,
	}

	fallback := func(ctx context.Context, key string) (interface{}, bool, error) {
		return "ttl_value", true, nil
	}

	var result string
	found, err := ts.Cacher.Get(ctx, "ttl_key", &result, fallback, opts)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "ttl_value", result)

	// 立即检查缓存命中
	fallbackCalled := false
	fallbackCheck := func(ctx context.Context, key string) (interface{}, bool, error) {
		fallbackCalled = true
		return "new_value", true, nil
	}

	found, err = ts.Cacher.Get(ctx, "ttl_key", &result, fallbackCheck, nil)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "ttl_value", result)
	assert.False(t, fallbackCalled) // 应该命中缓存

	// 等待TTL过期
	time.Sleep(150 * time.Millisecond)

	// 检查缓存是否过期
	found, err = ts.Cacher.Get(ctx, "ttl_key", &result, fallbackCheck, nil)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "new_value", result)
	assert.True(t, fallbackCalled) // 应该调用fallback，说明缓存过期了
}

// TestErrorHandling 测试错误处理
func (ts *CacherTestSuite) TestErrorHandling() {
	ctx := context.Background()
	t := ts.t

	// 测试批量fallback返回错误
	batchFallbackError := func(ctx context.Context, keys []string) (map[string]interface{}, error) {
		return nil, errors.New("batch fallback error")
	}

	keys := []string{"error_key1", "error_key2"}
	resultMap := make(map[string]string)
	err := ts.Cacher.MGet(ctx, keys, &resultMap, batchFallbackError, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "batch fallback error")

	// 测试空键列表
	err = ts.Cacher.MGet(ctx, []string{}, &resultMap, nil, nil)
	assert.NoError(t, err)

	// 测试删除空键列表
	deletedCount, err := ts.Cacher.MDelete(ctx, []string{})
	assert.NoError(t, err)
	assert.Equal(t, int64(0), deletedCount)
}

// TestCacherWithMockStore 使用MockStore测试Cacher实现
func TestCacherWithMockStore(t *testing.T) {
	mockStore := NewMockStore()
	cacher := NewCacher(mockStore)
	testSuite := NewCacherTestSuite(t, cacher)
	testSuite.RunAllTests()
	
	// 额外测试，验证MockStore工作正常
	ctx := context.Background()
	
	err := mockStore.MSet(ctx, map[string]interface{}{"test": "value"}, 0)
	assert.NoError(t, err)
	
	var result string
	found, err := mockStore.Get(ctx, "test", &result)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "value", result)
}
