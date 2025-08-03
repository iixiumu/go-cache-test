package ristretto

import (
	"testing"

	"go-cache/cacher/store"
)

// ristrettoStoreTester Ristretto Store测试器
type ristrettoStoreTester struct {
	store store.Store
}

func (r *ristrettoStoreTester) NewStore() (store.Store, error) {
	return NewRistrettoStore()
}

func (r *ristrettoStoreTester) Name() string {
	return "Ristretto"
}

func (r *ristrettoStoreTester) SetupTest(t *testing.T) {
	var err error
	r.store, err = NewRistrettoStore()
	if err != nil {
		t.Fatalf("failed to create ristretto store: %v", err)
	}
}

func (r *ristrettoStoreTester) TeardownTest(t *testing.T) {
	// Ristretto不需要特殊清理
}

func TestRistrettoStore(t *testing.T) {
	tester := &ristrettoStoreTester{}
	store.RunStoreTests(t, tester)
}

func TestRistrettoStoreSpecific(t *testing.T) {
	ristrettoStore, err := NewRistrettoStore()
	if err != nil {
		t.Fatalf("failed to create ristretto store: %v", err)
	}

	// Test various data types
	t.Run("TestString", func(t *testing.T) {
		key := "string_key"
		value := "test_string"

		err := ristrettoStore.MSet(nil, map[string]interface{}{key: value}, 0)
		if err != nil {
			t.Fatalf("MSet failed: %v", err)
		}

		var result string
		found, err := ristrettoStore.Get(nil, key, &result)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if !found {
			t.Error("Key should be found")
		}

		if result != value {
			t.Errorf("Expected %v, got %v", value, result)
		}
	})

	t.Run("TestInt", func(t *testing.T) {
		key := "int_key"
		value := 42

		err := ristrettoStore.MSet(nil, map[string]interface{}{key: value}, 0)
		if err != nil {
			t.Fatalf("MSet failed: %v", err)
		}

		var result int
		found, err := ristrettoStore.Get(nil, key, &result)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if !found {
			t.Error("Key should be found")
		}

		if result != value {
			t.Errorf("Expected %v, got %v", value, result)
		}
	})

	t.Run("TestStruct", func(t *testing.T) {
		type TestStruct struct {
			Name  string
			Value int
		}

		key := "struct_key"
		value := TestStruct{Name: "test", Value: 123}

		err := ristrettoStore.MSet(nil, map[string]interface{}{key: value}, 0)
		if err != nil {
			t.Fatalf("MSet failed: %v", err)
		}

		var result interface{}
		found, err := ristrettoStore.Get(nil, key, &result)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if !found {
			t.Error("Key should be found")
		}

		// 由于Ristretto直接存储对象，我们可以进行类型断言
		if testStruct, ok := result.(TestStruct); ok {
			if testStruct.Name != value.Name || testStruct.Value != value.Value {
				t.Errorf("Expected %+v, got %+v", value, testStruct)
			}
		}
	})
}
