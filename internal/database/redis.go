package database

import (
	"context"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

var RedisClient *redis.Client

func InitRedis() {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		log.Warn().Msg("REDIS_ADDR not set, skipping Redis init")
		return
	}

	RedisClient = redis.NewClient(&redis.Options{
		Addr: addr,
		DB:   0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if _, err := RedisClient.Ping(ctx).Result(); err != nil {
		log.Warn().
			Err(err).
			Msg("Redis unavailable, async jobs disabled")
		RedisClient = nil
	}
}
