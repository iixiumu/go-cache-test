package cacher

import (
	"testing"

	ristrettoV2 "github.com/dgraph-io/ristretto/v2"
	"go-cache/cacher/store/ristretto"
)

func TestCacherImpl(t *testing.T) {
	// 创建Ristretto缓存实例
	cache, err := ristrettoV2.NewCache(&ristrettoV2.Config[string, interface{}] {
		NumCounters: 1000,
		MaxCost:     100,
		BufferItems: 64,
	})
	if err != nil {
		t.Fatalf("Failed to create Ristretto cache: %v", err)
	}
	defer cache.Close()

	// 创建RistrettoStore实例
	ristrettoStore := ristretto.NewRistrettoStore(cache)

	// 创建CacherImpl实例
	cacher := NewCacherImpl(ristrettoStore)

	// 创建TestCacher实例
	tester := NewTestCacher(cacher)

	// 运行测试
	t.Run("Get", tester.TestGet)
	t.Run("MGet", tester.TestMGet)
	t.Run("MDelete", tester.TestMDelete)
	t.Run("MRefresh", tester.TestMRefresh)
	t.Run("CacheOptions", tester.TestCacheOptions)
}

// 测试使用Redis存储的Cacher实现
func TestCacherImplWithRedis(t *testing.T) {
	// 由于需要Redis服务器，这里只做简单测试
	// 在实际应用中，可以使用miniredis进行测试
	t.Skip("Skipping Redis test")
}
