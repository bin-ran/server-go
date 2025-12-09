package utils

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	CacheErrorString = "Cache Error"
)

// HSetAndExpireNonatomic set key and expire it.
// 本方法不考虑原子性，因此需要确保对该键设定的过期时间是一致的。
func HSetAndExpireNonatomic(client *redis.Client, ctx context.Context, key string, fields map[string]interface{}, expiration time.Duration) error {
	if err := client.HSet(ctx, key, fields).Err(); err != nil {
		return err
	}
	return client.Expire(ctx, key, expiration).Err()
}

// IncreaseAndExpireNonatomic increase key and expire it.
// 本方法不考虑原子性，因此需要确保对该键设定的过期时间是一致的。
func IncreaseAndExpireNonatomic(client *redis.Client, ctx context.Context, key string, expiration time.Duration) (int64, error) {
	res, err := client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	return res, client.Expire(ctx, key, expiration).Err()
}
