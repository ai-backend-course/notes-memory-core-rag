package database

import (
	"context"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

var Pool *pgxpool.Pool

func Connect() {
	log.Info().Msg("🔌 Starting database connection...")

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal().Msg("❌ DATABASE_URL is not set")
	}

	log.Info().Msg("🔌 Creating database connection pool...")

	// Context with timeout to avoid hanging connections
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatal().Err(err).Msg("❌ Failed to create database connection pool")
	}

	log.Info().Msg("🔌 Testing database connection...")

	if err := pool.Ping(ctx); err != nil {
		log.Fatal().Err(err).Msg("❌ Database ping failed")
	}

	log.Info().Msg("✅ Database connection established")

	Pool = pool

	// Run migrations with separate, longer context
	migrationCtx, migrationCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer migrationCancel()

	// ------------------------------
	// MIGRATIONS
	// ------------------------------
	log.Info().Msg("🔄 Running database migrations...")

	log.Info().Msg("🔄 Creating notes table...")
	_, err = pool.Exec(migrationCtx, `
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
	_, err = pool.Exec(migrationCtx, `
		CREATE EXTENSION IF NOT EXISTS vector;
	`)
	if err != nil {
		log.Fatal().Err(err).Msg("❌ Failed to enable pgvector extension")
	}

	_, err = pool.Exec(migrationCtx, `
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

	// Jobs table (async processing)
	_, err = pool.Exec(migrationCtx, `
		CREATE TABLE IF NOT EXISTS jobs (
			id UUID PRIMARY KEY,
			type TEXT NOT NULL,
			input JSONB NOT NULL,
			status TEXT NOT NULL CHECK (status IN ('queued', 'processing', 'completed', 'failed')),
			result JSONB,
			error TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	if err != nil {
		log.Fatal().Err(err).Msg("❌ Migration failed (jobs table)")
	}

	log.Info().Msg("🔄 Adding idempotency columns...")
	// Add new columns for idempotency (safe to run multiple times)
	_, err = pool.Exec(migrationCtx, `
		ALTER TABLE jobs 
		ADD COLUMN IF NOT EXISTS retry_count INTEGER DEFAULT 0;
	`)
	if err != nil {
		log.Fatal().Err(err).Msg("❌ Migration failed (retry_count column)")
	}

	_, err = pool.Exec(migrationCtx, `
		ALTER TABLE jobs 
		ADD COLUMN IF NOT EXISTS content_hash VARCHAR(128);
	`)
	if err != nil {
		log.Fatal().Err(err).Msg("❌ Migration failed (content_hash column)")
	}

	log.Info().Msg("🔄 Updating content_hash column size...")
	_, err = pool.Exec(migrationCtx, `
		ALTER TABLE jobs 
		ALTER COLUMN content_hash TYPE VARCHAR(128);
	`)
	if err != nil {
		log.Fatal().Err(err).Msg("❌ Migration failed (content_hash column resize)")
	}

	_, err = pool.Exec(migrationCtx, `
		CREATE INDEX IF NOT EXISTS idx_jobs_content_hash ON jobs(content_hash);
	`)
	if err != nil {
		log.Fatal().Err(err).Msg("❌ Migration failed (content_hash index)")
	}

	log.Info().Msg("🔄 Adding visibility timeout columns...")
	// Add columns for visibility timeout mechanism
	_, err = pool.Exec(migrationCtx, `
		ALTER TABLE jobs 
		ADD COLUMN IF NOT EXISTS visibility_timeout TIMESTAMPTZ,
		ADD COLUMN IF NOT EXISTS worker_id TEXT;
	`)
	if err != nil {
		log.Fatal().Err(err).Msg("❌ Migration failed (visibility timeout columns)")
	}

	_, err = pool.Exec(migrationCtx, `
		CREATE INDEX IF NOT EXISTS idx_jobs_visibility_timeout ON jobs(visibility_timeout) 
		WHERE visibility_timeout IS NOT NULL;
	`)
	if err != nil {
		log.Fatal().Err(err).Msg("❌ Migration failed (visibility timeout index)")
	}

	log.Info().Msg("✅ Database connected & migrations applied successfully")
}
