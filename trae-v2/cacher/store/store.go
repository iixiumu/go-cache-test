package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Store 底层存储接口，提供基础的键值存储操作
type Store interface {
	// Get 从存储后端获取单个值
	// key: 键名
	// dst: 目标变量的指针，用于接收反序列化后的值
	// 返回: 是否找到该键, 错误信息
	Get(ctx context.Context, key string, dst interface{}) (bool, error)

	// MGet 批量获取值到map中
	// keys: 要获取的键列表
	// dstMap: 目标map的指针，用于接收结果，类型为*map[string]T
	// 返回: 错误信息
	MGet(ctx context.Context, keys []string, dstMap interface{}) error

	// Exists 批量检查键存在性
	// keys: 要检查的键列表
	// 返回: map[string]bool 键存在性映射, 错误信息
	Exists(ctx context.Context, keys []string) (map[string]bool, error)

	// MSet 批量设置键值对，支持TTL
	// items: 键值对映射
	// ttl: 过期时间，0表示永不过期
	// 返回: 错误信息
	MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error

	// Del 删除指定键
	// keys: 要删除的键列表
	// 返回: 实际删除的键数量, 错误信息
	Del(ctx context.Context, keys ...string) (int64, error)
}

// StoreTestSuite 是一个通用的测试套件，用于测试任何实现了Store接口的存储后端
type StoreTestSuite struct {
	NewStore func() Store // 创建一个新的Store实例的函数
	Cleanup  func()      // 清理资源的函数
}

// RunTestSuite 运行所有Store接口的测试用例
func (s *StoreTestSuite) RunTestSuite(t *testing.T) {
	t.Run("TestGet", s.TestGet)
	t.Run("TestMGet", s.TestMGet)
	t.Run("TestExists", s.TestExists)
	t.Run("TestMSet", s.TestMSet)
	t.Run("TestDel", s.TestDel)
	t.Run("TestTTL", s.TestTTL)

	// 如果有清理函数，则执行
	if s.Cleanup != nil {
		defer s.Cleanup()
	}
}

// TestGet 测试Get方法
func (s *StoreTestSuite) TestGet(t *testing.T) {
	ctx := context.Background()
	store := s.NewStore()

	// 测试获取不存在的键
	var val string
	found, err := store.Get(ctx, "not_exist_key", &val)
	require.NoError(t, err)
	assert.False(t, found)

	// 设置一个字符串值并获取
	items := map[string]interface{}{
		"string_key": "hello world",
	}
	err = store.MSet(ctx, items, 0)
	require.NoError(t, err)

	found, err = store.Get(ctx, "string_key", &val)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "hello world", val)

	// 测试结构体值
	type TestStruct struct {
		Name string
		Age  int
	}

	original := TestStruct{Name: "test", Age: 18}
	items = map[string]interface{}{
		"struct_key": original,
	}
	err = store.MSet(ctx, items, 0)
	require.NoError(t, err)

	var retrieved TestStruct
	found, err = store.Get(ctx, "struct_key", &retrieved)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, original, retrieved)
}

// TestMGet 测试MGet方法
func (s *StoreTestSuite) TestMGet(t *testing.T) {
	ctx := context.Background()
	store := s.NewStore()

	// 设置多个值
	items := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	err := store.MSet(ctx, items, 0)
	require.NoError(t, err)

	// 测试批量获取
	result := make(map[string]string)
	err = store.MGet(ctx, []string{"key1", "key2", "not_exist"}, &result)
	require.NoError(t, err)
	assert.Equal(t, 2, len(result))
	assert.Equal(t, "value1", result["key1"])
	assert.Equal(t, "value2", result["key2"])
	assert.Empty(t, result["not_exist"])

	// 测试结构体批量获取
	type TestStruct struct {
		Name string
		Age  int
	}

	structItems := map[string]interface{}{
		"struct1": TestStruct{Name: "test1", Age: 18},
		"struct2": TestStruct{Name: "test2", Age: 20},
	}
	err = store.MSet(ctx, structItems, 0)
	require.NoError(t, err)

	structResult := make(map[string]TestStruct)
	err = store.MGet(ctx, []string{"struct1", "struct2", "not_exist"}, &structResult)
	require.NoError(t, err)
	assert.Equal(t, 2, len(structResult))
	assert.Equal(t, "test1", structResult["struct1"].Name)
	assert.Equal(t, 18, structResult["struct1"].Age)
	assert.Equal(t, "test2", structResult["struct2"].Name)
	assert.Equal(t, 20, structResult["struct2"].Age)
}

// TestExists 测试Exists方法
func (s *StoreTestSuite) TestExists(t *testing.T) {
	ctx := context.Background()
	store := s.NewStore()

	// 设置一些键
	items := map[string]interface{}{
		"exist_key1": "value1",
		"exist_key2": "value2",
	}
	err := store.MSet(ctx, items, 0)
	require.NoError(t, err)

	// 测试键存在性
	exists, err := store.Exists(ctx, []string{"exist_key1", "exist_key2", "not_exist_key"})
	require.NoError(t, err)
	assert.Equal(t, 3, len(exists))
	assert.True(t, exists["exist_key1"])
	assert.True(t, exists["exist_key2"])
	assert.False(t, exists["not_exist_key"])
}

// TestMSet 测试MSet方法
func (s *StoreTestSuite) TestMSet(t *testing.T) {
	ctx := context.Background()
	store := s.NewStore()

	// 测试设置多个值
	items := map[string]interface{}{
		"mset_key1": "value1",
		"mset_key2": 42,
		"mset_key3": true,
	}
	err := store.MSet(ctx, items, 0)
	require.NoError(t, err)

	// 验证设置的值
	var strVal string
	found, err := store.Get(ctx, "mset_key1", &strVal)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "value1", strVal)

	var intVal int
	found, err = store.Get(ctx, "mset_key2", &intVal)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, 42, intVal)

	var boolVal bool
	found, err = store.Get(ctx, "mset_key3", &boolVal)
	require.NoError(t, err)
	assert.True(t, found)
	assert.True(t, boolVal)

	// 测试结构体
	type TestStruct struct {
		Name string
		Age  int
	}

	structVal := TestStruct{Name: "test", Age: 18}
	items = map[string]interface{}{
		"struct_key": structVal,
	}
	err = store.MSet(ctx, items, 0)
	require.NoError(t, err)

	var retrievedStruct TestStruct
	found, err = store.Get(ctx, "struct_key", &retrievedStruct)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, structVal, retrievedStruct)
}

// TestDel 测试Del方法
func (s *StoreTestSuite) TestDel(t *testing.T) {
	ctx := context.Background()
	store := s.NewStore()

	// 设置一些键
	items := map[string]interface{}{
		"del_key1": "value1",
		"del_key2": "value2",
		"del_key3": "value3",
	}
	err := store.MSet(ctx, items, 0)
	require.NoError(t, err)

	// 删除部分键
	deleted, err := store.Del(ctx, "del_key1", "del_key2", "not_exist_key")
	require.NoError(t, err)
	assert.Equal(t, int64(2), deleted) // 应该删除了2个键

	// 验证删除结果
	exists, err := store.Exists(ctx, []string{"del_key1", "del_key2", "del_key3"})
	require.NoError(t, err)
	assert.False(t, exists["del_key1"])
	assert.False(t, exists["del_key2"])
	assert.True(t, exists["del_key3"])
}

// TestTTL 测试带TTL的键值存储
func (s *StoreTestSuite) TestTTL(t *testing.T) {
	ctx := context.Background()
	store := s.NewStore()

	// 设置键值对
	items := map[string]interface{}{
		"ttl_key": "will_expire",
	}
	err := store.MSet(ctx, items, 0) // 不设置TTL
	require.NoError(t, err)

	// 立即获取应该存在
	var val string
	found, err := store.Get(ctx, "ttl_key", &val)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "will_expire", val)

	// 手动删除键
	_, err = store.Del(ctx, "ttl_key")
	require.NoError(t, err)

	// 再次获取应该不存在
	found, err = store.Get(ctx, "ttl_key", &val)
	require.NoError(t, err)
	assert.False(t, found)
}
