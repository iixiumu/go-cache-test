package ristretto

import (
	"testing"
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/stretchr/testify/require"

	"go-cache/cacher/store"
)

func TestRistrettoStore(t *testing.T) {
	// 创建Ristretto存储实例
	ristrettoStore, err := New(Options{
		NumCounters: 1e4,
		MaxCost:     1e6,
		BufferItems: 64,
		Metrics:     true,
	})
	require.NoError(t, err)

	// 创建测试套件
	testSuite := &store.StoreTestSuite{
		NewStore: func() store.Store {
			return ristrettoStore
		},
		Cleanup: func() {
			// 清空Ristretto缓存
			ristrettoStore.cache.Clear()
			// 清空TTL映射
			ristrettoStore.mux.Lock()
			ristrettoStore.ttlMap = make(map[string]time.Time)
			ristrettoStore.mux.Unlock()
		},
	}

	// 运行所有测试
	testSuite.RunTestSuite(t)
}

func TestRistrettoStoreWithExistingCache(t *testing.T) {
	// 创建一个Ristretto缓存
	cache, err := ristretto.NewCache[string, []byte](&ristretto.Config[string, []byte]{
		NumCounters: 1e4,
		MaxCost:     1e6,
		BufferItems: 64,
	})
	require.NoError(t, err)

	// 使用现有缓存创建Ristretto存储实例
	ristrettoStore, err := New(Options{
		Cache: cache,
	})
	require.NoError(t, err)

	// 创建测试套件
	testSuite := &store.StoreTestSuite{
		NewStore: func() store.Store {
			return ristrettoStore
		},
		Cleanup: func() {
			// 清空Ristretto缓存
			ristrettoStore.cache.Clear()
			// 清空TTL映射
			ristrettoStore.mux.Lock()
			ristrettoStore.ttlMap = make(map[string]time.Time)
			ristrettoStore.mux.Unlock()
		},
	}

	// 运行所有测试
	testSuite.RunTestSuite(t)
}

func TestRistrettoStoreWithDefaultOptions(t *testing.T) {
	// 使用默认选项创建Ristretto存储实例
	ristrettoStore, err := New(Options{})
	require.NoError(t, err)

	// 创建测试套件
	testSuite := &store.StoreTestSuite{
		NewStore: func() store.Store {
			return ristrettoStore
		},
		Cleanup: func() {
			// 清空Ristretto缓存
			ristrettoStore.cache.Clear()
			// 清空TTL映射
			ristrettoStore.mux.Lock()
			ristrettoStore.ttlMap = make(map[string]time.Time)
			ristrettoStore.mux.Unlock()
		},
	}

	// 运行所有测试
	testSuite.RunTestSuite(t)
}