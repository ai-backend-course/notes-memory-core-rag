package database

import (
	"context"
	"os"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func InitRedis() {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	RedisClient = redis.NewClient(&redis.Options{
		Addr: addr,
		DB:   0,
	})

	ctx := context.Background()
	if _, err := RedisClient.Ping(ctx).Result(); err != nil {
		panic("Redis connection failed: " + err.Error())
	}
}
