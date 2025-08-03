package ristretto

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go-cache/cacher/store"
)

func TestRistrettoStore(t *testing.T) {
	// 创建Ristretto Store
	ristrettoStore, err := NewStore()
	require.NoError(t, err)
	defer ristrettoStore.Close()

	// 运行通用测试套件
	testHelper := store.NewTestHelper(t, ristrettoStore)
	testHelper.RunAllTests()
}
