package managers

import (
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var Redis *redis.Client

const (
	ACCOUNT = "A"
	IP      = "I"
	TOKEN   = "T"
)

const (
	IPLIMIT = "IP"
)

const (
	UserTokenLife = 1 * time.Hour
)

func InitRedis(wg *sync.WaitGroup) {
	options := redis.Options{
		Addr:     Config.Redis.URL,
		Password: Config.Redis.Password,
		DB:       Config.Redis.DB,
	}

	Redis = redis.NewClient(&options)
	slog.Info("Have connected to redis")
	wg.Done()
}
