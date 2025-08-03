package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go-cache/cacher"
	"go-cache/cacher/store/redis"
	"go-cache/cacher/store/ristretto"

	goredis "github.com/redis/go-redis/v9"
)

// User 示例用户结构体
type User struct {
	ID   int
	Name string
	Age  int
}

// 模拟数据库查询
func getUserFromDB(id int) (User, error) {
	// 在实际应用中，这里会查询数据库
	// 这里为了演示，直接返回模拟数据
	return User{
		ID:   id,
		Name: fmt.Sprintf("User-%d", id),
		Age:  20 + id,
	}, nil
}

// 模拟批量数据库查询
func getUsersFromDB(ids []int) (map[int]User, error) {
	result := make(map[int]User)
	for _, id := range ids {
		user, err := getUserFromDB(id)
		if err != nil {
			return nil, err
		}
		result[id] = user
	}
	return result, nil
}

func main() {
	// 创建上下文
	ctx := context.Background()

	// 示例1: 使用Redis作为存储后端
	fmt.Println("=== 使用Redis作为存储后端 ===")

	// 创建Redis客户端
	redisClient := goredis.NewClient(&goredis.Options{
		Addr: "localhost:6379", // 请确保Redis服务器正在运行
	})

	// 创建Redis存储
	redisStore, err := redis.New(redis.Options{
		Client: redisClient,
	})
	if err != nil {
		log.Fatalf("创建Redis存储失败: %v", err)
	}
	defer redisStore.Close()

	// 创建Redis缓存器
	redisCacher := cacher.NewDefaultCacher[User](cacher.DefaultCacherOptions{
		Store:      redisStore,
		Prefix:     "user",
		DefaultTTL: 5 * time.Minute,
	})

	// 使用Redis缓存器获取用户
	userID := 1
	var user User
	found, err := redisCacher.Get(ctx, fmt.Sprintf("%d", userID), &user, func(ctx context.Context, key string) (interface{}, bool, error) {
		fmt.Println("从数据库获取用户(Redis)")
		u, err := getUserFromDB(userID)
		return u, err == nil, err
	}, nil)
	if err != nil {
		log.Printf("获取用户失败: %v", err)
	} else if found {
		fmt.Printf("获取到用户: %+v\n", user)
	} else {
		fmt.Println("未找到用户")
	}

	// 再次获取同一用户，应该从缓存中获取
	var cachedUser User
	found, err = redisCacher.Get(ctx, fmt.Sprintf("%d", userID), &cachedUser, func(ctx context.Context, key string) (interface{}, bool, error) {
		fmt.Println("从数据库获取用户(Redis)")
		u, err := getUserFromDB(userID)
		return u, err == nil, err
	}, nil)
	if err != nil {
		log.Printf("获取用户失败: %v", err)
	} else if found {
		fmt.Printf("再次获取到用户(应该从缓存): %+v\n", cachedUser)
	} else {
		fmt.Println("未找到用户")
	}

	// 批量获取用户
	userIDs := []int{2, 3, 4}
	userKeys := make([]string, len(userIDs))
	for i, id := range userIDs {
		userKeys[i] = fmt.Sprintf("%d", id)
	}

	// 创建结果map
	usersMap := make(map[string]*User)
	for _, key := range userKeys {
		usersMap[key] = &User{}
	}

	foundKeys, err := redisCacher.MGet(ctx, userKeys, usersMap, func(ctx context.Context, keys []string) (map[string]interface{}, []string, error) {
		fmt.Println("批量从数据库获取用户(Redis)")
		// 将字符串键转换为整数ID
		ids := make([]int, len(keys))
		for i, key := range keys {
			var id int
			_, err := fmt.Sscanf(key, "%d", &id)
			if err != nil {
				return nil, nil, err
			}
			ids[i] = id
		}

		// 从数据库获取用户
		idUsers, err := getUsersFromDB(ids)
		if err != nil {
			return nil, nil, err
		}

		// 将整数ID映射回字符串键
		result := make(map[string]interface{})
		foundKeys := make([]string, 0, len(keys))
		for i, key := range keys {
			result[key] = idUsers[ids[i]]
			foundKeys = append(foundKeys, key)
		}

		return result, foundKeys, nil
	}, nil)
	if err != nil {
		log.Printf("批量获取用户失败: %v", err)
	} else {
		fmt.Printf("批量获取到用户，找到 %d 个: %+v\n", len(foundKeys), usersMap)
	}

	// 示例2: 使用Ristretto作为存储后端
	fmt.Println("\n=== 使用Ristretto作为存储后端 ===")

	// 创建Ristretto存储
	ristrettoStore, err := ristretto.New(ristretto.Options{
		NumCounters: 1e7,     // 预期缓存项数量的10倍
		MaxCost:     1 << 30, // 最大内存使用量: 1GB
		BufferItems: 64,      // 缓冲区大小
	})
	if err != nil {
		log.Fatalf("创建Ristretto存储失败: %v", err)
	}
	defer ristrettoStore.Close()

	// 创建Ristretto缓存器
	ristrettoCacher := cacher.NewDefaultCacher[User](cacher.DefaultCacherOptions{
		Store:      ristrettoStore,
		Prefix:     "user",
		DefaultTTL: 5 * time.Minute,
	})

	// 使用Ristretto缓存器获取用户
	userID = 5
	var ristrettoUser User
	found, err = ristrettoCacher.Get(ctx, fmt.Sprintf("%d", userID), &ristrettoUser, func(ctx context.Context, key string) (interface{}, bool, error) {
		fmt.Println("从数据库获取用户(Ristretto)")
		u, err := getUserFromDB(userID)
		return u, err == nil, err
	}, nil)
	if err != nil {
		log.Printf("获取用户失败: %v", err)
	} else if found {
		fmt.Printf("获取到用户: %+v\n", ristrettoUser)
	} else {
		fmt.Println("未找到用户")
	}

	// 再次获取同一用户，应该从缓存中获取
	var cachedRistrettoUser User
	found, err = ristrettoCacher.Get(ctx, fmt.Sprintf("%d", userID), &cachedRistrettoUser, func(ctx context.Context, key string) (interface{}, bool, error) {
		fmt.Println("从数据库获取用户(Ristretto)")
		u, err := getUserFromDB(userID)
		return u, err == nil, err
	}, nil)
	if err != nil {
		log.Printf("获取用户失败: %v", err)
	} else if found {
		fmt.Printf("再次获取到用户(应该从缓存): %+v\n", cachedRistrettoUser)
	} else {
		fmt.Println("未找到用户")
	}

	// 删除缓存
	fmt.Println("\n=== 删除缓存 ===")
	deleted, err := ristrettoCacher.MDelete(ctx, []string{fmt.Sprintf("%d", userID)})
	if err != nil {
		log.Printf("删除缓存失败: %v", err)
	} else {
		fmt.Printf("缓存已删除，删除了 %d 个键\n", deleted)
	}

	// 再次获取用户，应该从数据库获取
	var deletedUser User
	found, err = ristrettoCacher.Get(ctx, fmt.Sprintf("%d", userID), &deletedUser, func(ctx context.Context, key string) (interface{}, bool, error) {
		fmt.Println("从数据库获取用户(Ristretto)")
		u, err := getUserFromDB(userID)
		return u, err == nil, err
	}, nil)
	if err != nil {
		log.Printf("获取用户失败: %v", err)
	} else if found {
		fmt.Printf("删除缓存后再次获取到用户: %+v\n", deletedUser)
	} else {
		fmt.Println("未找到用户")
	}

	// 刷新缓存
	fmt.Println("\n=== 刷新缓存 ===")
	// 创建结果map
	refreshMap := make(map[string]*User)
	refreshMap[fmt.Sprintf("%d", userID)] = &User{}

	err = ristrettoCacher.MRefresh(ctx, []string{fmt.Sprintf("%d", userID)}, &refreshMap, func(ctx context.Context, keys []string) (map[string]interface{}, []string, error) {
		fmt.Println("刷新缓存，从数据库获取最新数据")
		// 将字符串键转换为整数ID
		ids := make([]int, len(keys))
		for i, key := range keys {
			var id int
			_, err := fmt.Sscanf(key, "%d", &id)
			if err != nil {
				return nil, nil, err
			}
			ids[i] = id
		}

		// 从数据库获取用户
		idUsers, err := getUsersFromDB(ids)
		if err != nil {
			return nil, nil, err
		}

		// 将整数ID映射回字符串键
		result := make(map[string]interface{})
		foundKeys := make([]string, 0, len(keys))
		for i, key := range keys {
			result[key] = idUsers[ids[i]]
			foundKeys = append(foundKeys, key)
		}

		return result, foundKeys, nil
	}, nil)
	if err != nil {
		log.Printf("刷新缓存失败: %v", err)
	} else {
		fmt.Println("缓存已刷新")
	}

	// 获取刷新后的用户，应该从缓存中获取
	var refreshedUser User
	found, err = ristrettoCacher.Get(ctx, fmt.Sprintf("%d", userID), &refreshedUser, func(ctx context.Context, key string) (interface{}, bool, error) {
		fmt.Println("从数据库获取用户(Ristretto)")
		u, err := getUserFromDB(userID)
		return u, err == nil, err
	}, nil)
	if err != nil {
		log.Printf("获取用户失败: %v", err)
	} else if found {
		fmt.Printf("刷新缓存后获取到用户: %+v\n", refreshedUser)
	} else {
		fmt.Println("未找到用户")
	}
}