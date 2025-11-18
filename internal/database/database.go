package database

import (
	"context"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// Pool is the shared PostgreSQL connection pool.
var Pool *pgxpool.Pool

// Connect initializes the database connection pool and applies
// migrations for both the notes and embedding tables.
func Connect() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal().Msg("❌ DATABASE_URL is not set")
	}

	// Context with timeout to avoid hanging connections
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create connection pool
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatal().Err(err).Msg("❌ Failed to create database connection pool")
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		log.Fatal().Err(err).Msg("❌ Database ping failed")
	}

	Pool = pool

	// ------------------------------
	// MIGRATIONS
	// ------------------------------

	// Notes table
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS notes (
			id SERIAL PRIMARY KEY,
			title TEXT NOT NULL,
			content TEXT,
			created_at TIMESTAMP DEFAULT NOW()
		);
	`)
	if err != nil {
		log.Fatal().Err(err).Msg("❌ Migration failed (notes table)")
	}

	// Enable pgvector extension
	_, err = pool.Exec(ctx, `
		CREATE EXTENSION IF NOT EXISTS vector;
	`)
	if err != nil {
		log.Fatal().Err(err).Msg("❌ Failed to enable pgvector extension")
	}

	// Embeddings table (1536-dim vector)
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS note_embeddings (
			id SERIAL PRIMARY KEY,
			note_id INTEGER REFERENCES notes(id) ON DELETE CASCADE,
			embedding vector(1536),
			created_at TIMESTAMP DEFAULT NOW()
		);
	`)
	if err != nil {
		log.Fatal().Err(err).Msg("❌ Migration failed (note_embeddings table)")
	}

	log.Info().Msg("✅ Database connected & migrations applied successfully")
}
