package ristretto

import (
	"testing"

	"go-cache/cacher/store"
)

// TestRistrettoStore 测试RistrettoStore实现
func TestRistrettoStore(t *testing.T) {
	// 测试Store接口
	store.StoreTestHelper(t, func() (store.Store, func(), error) {
		// 创建RistrettoStore实例
		ristrettoStore, err := NewRistrettoStore(1000)
		if err != nil {
			t.Fatalf("创建RistrettoStore失败: %v", err)
		}

		// 返回Store实例和清理函数
		return ristrettoStore, func() {
			// Ristretto没有明确的关闭方法
		}, nil
	})
}