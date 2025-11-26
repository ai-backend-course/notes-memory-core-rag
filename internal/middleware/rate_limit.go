package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/storage/memory"
)

func RateLimit() fiber.Handler {
	store := memory.New()

	return limiter.New(limiter.Config{
		Max:        3,
		Expiration: 1 * time.Minute,
		Store:      store,
	})
}
