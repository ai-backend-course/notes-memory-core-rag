package main

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"notes-memory-core-rag/internal/database"
	"notes-memory-core-rag/internal/handlers"
	"notes-memory-core-rag/internal/middleware"
)

func main() {
	// Pretty logger output during development
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Warn().Msg("No .env file found, using system environment variables")
	}

	// Connect to Postgres and run migrations
	database.Connect()

	// Create Fiber app
	app := fiber.New()

	// CORS enabled
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,OPTIONS",
	}))

	// Middleware
	app.Use(middleware.MetricsMiddleware)
	app.Use(middleware.LoggerMiddleware)
	app.Use(middleware.RateLimit())

	// Base routes (health + notes CRUD)
	app.Get("/health", handlers.HealthCheck)
	app.Get("/notes", handlers.GetNotes)
	app.Post("/notes", handlers.CreateNote)

	// RAG routes
	app.Post("/query", handlers.Query)           // full RAG answer
	app.Post("/search", handlers.SemanticSearch) // vector search only

	// Metrics endpoint
	app.Get("/metrics", func(c *fiber.Ctx) error {
		return c.JSON(middleware.GetMetrics())
	})

	// Port
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Info().
		Str("port", port).
		Msg("🚀 Starting RAG Notes API")

	if err := app.Listen(":" + port); err != nil {
		log.Fatal().Err(err).Msg("❌ Server failed to start")
	}
}
