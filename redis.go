package main

import (
	"context"
	"os"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

func NewRedisClient() *redis.Client {
	addr := os.Getenv("REDIS_ADDR")
	return redis.NewClient(&redis.Options{
		Addr: addr,
		DB:   0,
	})
}
