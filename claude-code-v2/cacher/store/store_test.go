package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSuite 统一的Store接口测试套件
type TestSuite struct {
	Store Store
	t     *testing.T
}

// NewTestSuite 创建新的测试套件
func NewTestSuite(t *testing.T, store Store) *TestSuite {
	return &TestSuite{
		Store: store,
		t:     t,
	}
}

// RunAllTests 执行所有Store接口测试
func (ts *TestSuite) RunAllTests() {
	ts.TestGet()
	ts.TestMGet()
	ts.TestExists()
	ts.TestMSet()
	ts.TestDel()
	ts.TestTTL()
	ts.TestEdgeCases()
}

// TestGet 测试Get方法
func (ts *TestSuite) TestGet() {
	ctx := context.Background()
	t := ts.t

	// 测试获取不存在的键
	var result string
	found, err := ts.Store.Get(ctx, "nonexistent", &result)
	assert.NoError(t, err)
	assert.False(t, found)

	// 测试设置和获取字符串值
	err = ts.Store.MSet(ctx, map[string]interface{}{"test_key": "test_value"}, 0)
	require.NoError(t, err)

	found, err = ts.Store.Get(ctx, "test_key", &result)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "test_value", result)

	// 测试获取不同类型的值
	var intResult int
	err = ts.Store.MSet(ctx, map[string]interface{}{"int_key": 42}, 0)
	require.NoError(t, err)

	found, err = ts.Store.Get(ctx, "int_key", &intResult)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, 42, intResult)

	// 测试获取复杂结构体
	type TestStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	
	original := TestStruct{Name: "Alice", Age: 30}
	err = ts.Store.MSet(ctx, map[string]interface{}{"struct_key": original}, 0)
	require.NoError(t, err)

	var structResult TestStruct
	found, err = ts.Store.Get(ctx, "struct_key", &structResult)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, original, structResult)
}

// TestMGet 测试MGet方法
func (ts *TestSuite) TestMGet() {
	ctx := context.Background()
	t := ts.t

	// 设置测试数据
	testData := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	err := ts.Store.MSet(ctx, testData, 0)
	require.NoError(t, err)

	// 测试获取存在的键
	keys := []string{"key1", "key2", "key3"}
	resultMap := make(map[string]string)
	err = ts.Store.MGet(ctx, keys, &resultMap)
	assert.NoError(t, err)
	assert.Len(t, resultMap, 3)
	assert.Equal(t, "value1", resultMap["key1"])
	assert.Equal(t, "value2", resultMap["key2"])
	assert.Equal(t, "value3", resultMap["key3"])

	// 测试部分键存在的情况
	mixedKeys := []string{"key1", "nonexistent", "key3"}
	mixedResultMap := make(map[string]string)
	err = ts.Store.MGet(ctx, mixedKeys, &mixedResultMap)
	assert.NoError(t, err)
	assert.Len(t, mixedResultMap, 2)
	assert.Equal(t, "value1", mixedResultMap["key1"])
	assert.Equal(t, "value3", mixedResultMap["key3"])
	_, exists := mixedResultMap["nonexistent"]
	assert.False(t, exists)

	// 测试空键列表
	emptyResultMap := make(map[string]string)
	err = ts.Store.MGet(ctx, []string{}, &emptyResultMap)
	assert.NoError(t, err)
	assert.Len(t, emptyResultMap, 0)
}

// TestExists 测试Exists方法
func (ts *TestSuite) TestExists() {
	ctx := context.Background()
	t := ts.t

	// 设置测试数据
	testData := map[string]interface{}{
		"exist1": "value1",
		"exist2": "value2",
	}
	err := ts.Store.MSet(ctx, testData, 0)
	require.NoError(t, err)

	// 测试存在性检查
	keys := []string{"exist1", "nonexistent", "exist2"}
	existsMap, err := ts.Store.Exists(ctx, keys)
	assert.NoError(t, err)
	assert.Len(t, existsMap, 3)
	assert.True(t, existsMap["exist1"])
	assert.False(t, existsMap["nonexistent"])
	assert.True(t, existsMap["exist2"])

	// 测试空键列表
	emptyExistsMap, err := ts.Store.Exists(ctx, []string{})
	assert.NoError(t, err)
	assert.Len(t, emptyExistsMap, 0)
}

// TestMSet 测试MSet方法
func (ts *TestSuite) TestMSet() {
	ctx := context.Background()
	t := ts.t

	// 测试基本设置
	testData := map[string]interface{}{
		"mset1": "value1",
		"mset2": 42,
		"mset3": true,
	}
	err := ts.Store.MSet(ctx, testData, 0)
	assert.NoError(t, err)

	// 验证设置成功
	var strResult string
	found, err := ts.Store.Get(ctx, "mset1", &strResult)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "value1", strResult)

	var intResult int
	found, err = ts.Store.Get(ctx, "mset2", &intResult)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, 42, intResult)

	var boolResult bool
	found, err = ts.Store.Get(ctx, "mset3", &boolResult)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, true, boolResult)

	// 测试空map
	err = ts.Store.MSet(ctx, map[string]interface{}{}, 0)
	assert.NoError(t, err)
}

// TestDel 测试Del方法
func (ts *TestSuite) TestDel() {
	ctx := context.Background()
	t := ts.t

	// 设置测试数据
	testData := map[string]interface{}{
		"del1": "value1",
		"del2": "value2",
		"del3": "value3",
	}
	err := ts.Store.MSet(ctx, testData, 0)
	require.NoError(t, err)

	// 测试删除存在的键
	deletedCount, err := ts.Store.Del(ctx, "del1", "del2")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), deletedCount)

	// 验证键已被删除
	var result string
	found, err := ts.Store.Get(ctx, "del1", &result)
	assert.NoError(t, err)
	assert.False(t, found)

	found, err = ts.Store.Get(ctx, "del2", &result)
	assert.NoError(t, err)
	assert.False(t, found)

	// 验证未删除的键仍然存在
	found, err = ts.Store.Get(ctx, "del3", &result)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "value3", result)

	// 测试删除不存在的键
	deletedCount, err = ts.Store.Del(ctx, "nonexistent1", "nonexistent2")
	assert.NoError(t, err)
	assert.Equal(t, int64(0), deletedCount)

	// 测试混合删除（存在和不存在的键）
	deletedCount, err = ts.Store.Del(ctx, "del3", "nonexistent")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), deletedCount)
}

// TestTTL 测试TTL功能
func (ts *TestSuite) TestTTL() {
	ctx := context.Background()
	t := ts.t

	// 测试设置带TTL的键
	testData := map[string]interface{}{
		"ttl_key": "ttl_value",
	}
	err := ts.Store.MSet(ctx, testData, 100*time.Millisecond)
	require.NoError(t, err)

	// 立即检查键是否存在
	var result string
	found, err := ts.Store.Get(ctx, "ttl_key", &result)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "ttl_value", result)

	// 等待TTL过期
	time.Sleep(150 * time.Millisecond)

	// 检查键是否已过期
	found, err = ts.Store.Get(ctx, "ttl_key", &result)
	assert.NoError(t, err)
	assert.False(t, found)
}

// TestEdgeCases 测试边界情况
func (ts *TestSuite) TestEdgeCases() {
	ctx := context.Background()
	t := ts.t

	// 测试nil值
	err := ts.Store.MSet(ctx, map[string]interface{}{"nil_key": nil}, 0)
	assert.NoError(t, err)

	var result interface{}
	found, err := ts.Store.Get(ctx, "nil_key", &result)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Nil(t, result)

	// 测试空字符串键
	err = ts.Store.MSet(ctx, map[string]interface{}{"": "empty_key_value"}, 0)
	assert.NoError(t, err)

	var emptyKeyResult string
	found, err = ts.Store.Get(ctx, "", &emptyKeyResult)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "empty_key_value", emptyKeyResult)

	// 测试很长的键名
	longKey := string(make([]byte, 1000))
	for i := range longKey {
		longKey = longKey[:i] + "a" + longKey[i+1:]
	}
	err = ts.Store.MSet(ctx, map[string]interface{}{longKey: "long_key_value"}, 0)
	assert.NoError(t, err)

	var longKeyResult string
	found, err = ts.Store.Get(ctx, longKey, &longKeyResult)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "long_key_value", longKeyResult)

	// 测试大量数据
	largeData := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		largeData[string(rune('a'+i%26))+string(rune('0'+i%10))] = i
	}
	err = ts.Store.MSet(ctx, largeData, 0)
	assert.NoError(t, err)

	// 验证部分数据
	var intResult int
	found, err = ts.Store.Get(ctx, "a0", &intResult)
	assert.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, 0, intResult)
}