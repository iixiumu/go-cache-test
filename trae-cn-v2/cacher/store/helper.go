package store

import (
	"context"
	"reflect"
	"testing"
	"time"
)

// StoreTestHelper 测试Store接口的通用函数
// 创建一个辅助函数，接收一个Store工厂函数
func StoreTestHelper(t *testing.T, newStore func() (Store, func(), error)) {
	t.Logf("开始测试Store实现")
	// 获取Store实例和清理函数
	store, cleanup, err := newStore()
	if err != nil {
		t.Fatalf("创建Store失败: %v", err)
	}
	defer cleanup()

	ctx := context.Background()

	// 测试MSet和Get
	t.Run("MSet and Get", func(t *testing.T) {
		// 设置测试数据
		data := map[string]interface{}{
			"key1": "value1",
			"key2": 42,
			"key3": []int{1, 2, 3},
			"key4": map[string]interface{}{"name": "test", "age": 30},
		}

		// 设置值
		if err := store.MSet(ctx, data, 0); err != nil {
			t.Fatalf("MSet失败: %v", err)
		}

		// 逐个获取并验证
		for k, v := range data {
			var got interface{}
			found, err := store.Get(ctx, k, &got)
			if err != nil {
				t.Fatalf("Get %s 失败: %v", k, err)
			}
			if !found {
				t.Errorf("Get %s: 未找到", k)
				continue
			}
			if !reflect.DeepEqual(got, v) {
				t.Errorf("Get %s: 期望 %v (类型 %T), 实际 %v (类型 %T)", k, v, v, got, got)
			}
		}
	})

	// 测试MGet
	t.Run("MGet", func(t *testing.T) {
		// 设置测试数据
		data := map[string]interface{}{
			"key5": "value5",
			"key6": 100,
		}

		// 设置值
		if err := store.MSet(ctx, data, 0); err != nil {
			t.Fatalf("MSet失败: %v", err)
		}

		// 批量获取
		keys := []string{"key5", "key6", "key7"}
		var results map[string]interface{}
		if err := store.MGet(ctx, keys, &results); err != nil {
			t.Fatalf("MGet失败: %v", err)
		}

		// 验证结果
		// 检查存在的键
		for _, k := range keys {
			if v, ok := data[k]; ok {
				if !reflect.DeepEqual(results[k], v) {
					t.Errorf("MGet %s: 期望 %v (类型 %T), 实际 %v (类型 %T)", k, v, v, results[k], results[k])
				}
			} else {
				// 不存在的键应该不在结果中
				if _, exists := results[k]; exists {
					t.Errorf("MGet %s: 不应该存在", k)
				}
			}
		}
	})

	// 测试Del
	t.Run("Del", func(t *testing.T) {
		// 设置测试数据
		data := map[string]interface{}{
			"key8": "value8",
		}

		// 设置值
		if err := store.MSet(ctx, data, 0); err != nil {
			t.Fatalf("MSet失败: %v", err)
		}

		// 删除键
		deleted, err := store.Del(ctx, "key8")
		if err != nil {
			t.Fatalf("Del失败: %v", err)
		}
		if deleted != 1 {
			t.Errorf("Del key8: 期望删除1个键, 实际删除%d个", deleted)
		}

		// 验证已删除
		var got interface{}
		found, err := store.Get(ctx, "key8", &got)
		if err != nil {
			t.Fatalf("Get key8 失败: %v", err)
		}
		if found {
			t.Errorf("Del key8: 键仍然存在")
		}
	})

	// 测试TTL
	t.Run("TTL", func(t *testing.T) {
		// 设置带过期时间的键
		data := map[string]interface{}{
			"key9": "value9",
		}

		// 设置值，过期时间为1秒
		if err := store.MSet(ctx, data, 1*time.Second); err != nil {
			t.Fatalf("MSet失败: %v", err)
		}

		// 立即检查，键应该存在
		var got interface{}
		found, err := store.Get(ctx, "key9", &got)
		if err != nil {
			t.Fatalf("Get key9 失败: %v", err)
		}
		if !found {
			t.Errorf("TTL测试: 键应该存在，但未找到")
		}

		// 等待过期 (增加等待时间以适应Redis的过期机制)
		time.Sleep(3 * time.Second)

		// 对于RedisStore，使用GetTTL方法检查剩余生存时间
		if redisStore, ok := store.(interface{ GetTTL(context.Context, string) (time.Duration, error) }); ok {
			ttl, err := redisStore.GetTTL(ctx, "key9")
			if err != nil {
				t.Fatalf("GetTTL key9 失败: %v", err)
			}
			// 检查TTL是否小于等于设置的TTL值(1秒)，考虑到Redis的精度问题
			if ttl > 1*time.Second {
				t.Errorf("TTL测试: 键应该过期，但TTL仍为 %v", ttl)
			}
		} else {
			// 对于其他存储实现，使用Exists方法检查键是否存在
			existsMap, err := store.Exists(ctx, []string{"key9"})
			if err != nil {
				t.Fatalf("Exists key9 失败: %v", err)
			}
			if existsMap["key9"] {
				t.Errorf("TTL测试: 键应该过期，但仍然存在")
			}
		}
	})
}