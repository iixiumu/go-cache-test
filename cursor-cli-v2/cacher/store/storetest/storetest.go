package storetest

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"go-cache/cacher/store"
)

type NewStoreFunc func(t *testing.T) store.Store

// Run executes a common suite of tests against a Store implementation.
func Run(t *testing.T, newStore NewStoreFunc) {
	t.Run("Get_MSet_BasicTypes", func(t *testing.T) {
		st := newStore(t)
		ctx := context.Background()

		items := map[string]interface{}{
			"k_str": "hello",
			"k_int": 42,
			"k_struct": struct {
				Name string
				Age  int
			}{Name: "alice", Age: 30},
		}
		requireNoError(t, st.MSet(ctx, items, 0))

		// string
		var s string
		ok, err := st.Get(ctx, "k_str", &s)
		requireNoError(t, err)
		if !ok || s != "hello" {
			t.Fatalf("expected hit and value 'hello', got ok=%v value=%v", ok, s)
		}

		// int
		var i int
		ok, err = st.Get(ctx, "k_int", &i)
		requireNoError(t, err)
		if !ok || i != 42 {
			t.Fatalf("expected hit and value 42, got ok=%v value=%v", ok, i)
		}

		// struct
		var u struct {
			Name string
			Age  int
		}
		ok, err = st.Get(ctx, "k_struct", &u)
		requireNoError(t, err)
		if !ok || u.Name != "alice" || u.Age != 30 {
			t.Fatalf("unexpected struct: ok=%v, val=%+v", ok, u)
		}
	})

	t.Run("MGet_Partial", func(t *testing.T) {
		st := newStore(t)
		ctx := context.Background()

		requireNoError(t, st.MSet(ctx, map[string]interface{}{
			"a": 1,
			"b": 2,
		}, 0))

		var out map[string]int
		err := st.MGet(ctx, []string{"a", "b", "c"}, &out)
		requireNoError(t, err)
		if len(out) != 2 || out["a"] != 1 || out["b"] != 2 {
			t.Fatalf("unexpected MGet result: %+v", out)
		}
		if _, ok := out["c"]; ok {
			t.Fatalf("key 'c' should not exist in result")
		}
	})

	t.Run("Exists_And_Del", func(t *testing.T) {
		st := newStore(t)
		ctx := context.Background()

		requireNoError(t, st.MSet(ctx, map[string]interface{}{
			"x": "vx",
			"y": "vy",
		}, 0))

		exists, err := st.Exists(ctx, []string{"x", "y", "z"})
		requireNoError(t, err)
		if !exists["x"] || !exists["y"] || exists["z"] {
			t.Fatalf("unexpected exists: %+v", exists)
		}

		deln, err := st.Del(ctx, "x", "z")
		requireNoError(t, err)
		if deln != 1 {
			t.Fatalf("expected delete count 1, got %d", deln)
		}

		exists, err = st.Exists(ctx, []string{"x", "y"})
		requireNoError(t, err)
		if exists["x"] || !exists["y"] {
			t.Fatalf("unexpected exists after del: %+v", exists)
		}
	})

	t.Run("TTL_Expiry", func(t *testing.T) {
		st := newStore(t)
		ctx := context.Background()

		requireNoError(t, st.MSet(ctx, map[string]interface{}{"ttl": 100}, 50*time.Millisecond))
		var v int
		ok, err := st.Get(ctx, "ttl", &v)
		requireNoError(t, err)
		if !ok || v != 100 {
			t.Fatalf("unexpected get before expire: ok=%v v=%v", ok, v)
		}
		// Try to advance time in a store-specific way if supported
		if ff, ok := any(st).(interface{ TestFastForward(time.Duration) }); ok {
			ff.TestFastForward(80 * time.Millisecond)
		} else {
			time.Sleep(80 * time.Millisecond)
		}
		ok, err = st.Get(ctx, "ttl", &v)
		requireNoError(t, err)
		if ok {
			t.Fatalf("expected expired, still found")
		}
	})

	t.Run("TypeSafety_MapDestinationMustBePointerToMap", func(t *testing.T) {
		st := newStore(t)
		ctx := context.Background()
		requireNoError(t, st.MSet(ctx, map[string]interface{}{"a": 1}, 0))
		var notPtr map[string]int
		err := st.MGet(ctx, []string{"a"}, notPtr)
		if err == nil {
			t.Fatalf("expected error when dstMap is not a pointer")
		}
		var wrongType map[string]string
		err = st.MGet(ctx, []string{"a"}, &wrongType)
		if err == nil {
			t.Fatalf("expected error when dstMap element type mismatches")
		}

		// Ensure redis JSON numbers decode into numbers properly (via json number handling)
		var okType map[string]int
		requireNoError(t, st.MGet(ctx, []string{"a"}, &okType))
		if okType["a"] != 1 {
			t.Fatalf("unexpected decoded value: %+v", okType)
		}
	})
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Helper to decode JSON into a typed value for comparison in some stores if needed.
func decodeJSONTo(value []byte, dst interface{}) error {
	return json.Unmarshal(value, dst)
}
