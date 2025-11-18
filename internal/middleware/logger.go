package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
)

// LoggerMiddleware logs incoming HTTP requests with method, path, status,
// and response duration. This matches production logging patterns.
func LoggerMiddleware(c *fiber.Ctx) error {
	start := time.Now()

	err := c.Next()

	duration := time.Since(start)

	log.Info().
		Str("method", c.Method()).
		Str("path", c.Path()).
		Int("status", c.Response().StatusCode()).
		Dur("duration_ms", duration).
		Msg("request handled")

	return err
}
