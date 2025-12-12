# Notes Memory Core — RAG Extension

![Go Version](https://img.shields.io/badge/Go-1.23-blue)
![Fiber](https://img.shields.io/badge/Framework-Fiber%20v2-forestgreen)
![Postgres](https://img.shields.io/badge/Database-Postgres%2016-blue)
![License](https://img.shields.io/badge/License-MIT-green)
![Build](https://img.shields.io/badge/Build-Passing-brightgreen)
![Status](https://img.shields.io/badge/Project-RAG%20API%20Running-blue)
![CI](https://github.com/ai-backend-course/notes-memory-core-rag/actions/workflows/ci.yml/badge.svg)
![Deploy](https://github.com/ai-backend-course/notes-memory-core-rag/actions/workflows/fly-deploy.yml/badge.svg)




A production-ready Go backend demonstrating Retrieval-Augmented Generation (RAG) using:

- Go 1.23
- Fiber v2
- Postgres 16 + pgvector
- OpenAI embeddings (SmallEmbedding3)
- GPT-4o Mini for generation
- Optional mock AI mode (no API key required)
- Docker + Docker Compose

This repository extends the base notes-memory-core backend into a full AI retrieval system:

- Store notes
- Generate vector embeddings
- Run semantic search using pgvector
- Produce AI answers grounded in your notes (RAG)

The system supports **both synchronous and asynchronous execution modes**, with graceful degradation when optional infrastructure is unavailable.

---

## 🚀 Features

### Core Backend
- CRUD Notes API
- Structured logging (zerolog)
- In-memory metrics at /metrics
- Automatic migrations
- Dockerized Postgres 16
- Rate limiting middleware to protect AI-backed endpoints


### RAG Features
- pgvector semantic search
- Embeddings: mock OR real OpenAI
- LLM responses: mock OR real OpenAI
- Clean modular AI architecture
- Fully runnable without any API keys

---

## 📂 Project Structure

```text
notes-memory-core-rag/
├── main.go                     # API entrypoint
├── Dockerfile
├── docker-compose.yml
├── fly.toml
├── .env.example
├── README.md
│
├── cmd/
│   └── worker/
│       └── main.go             # Background job worker (Redis-based)
│
├── internal/
│   ├── ai/                     # AI abstraction layer
│   │   ├── embeddings.go       # Mock + real embeddings (ctx-aware)
│   │   ├── responder.go        # Mock + real LLM responses
│   │   └── openai.go
│   │
│   ├── database/
│   │   ├── database.go         # Postgres + migrations
│   │   ├── redis.go            # Optional Redis initialization
│   │   └── jobs.go             # Async job persistence
│   │
│   ├── handlers/
│   │   ├── notes.go            # CRUD notes
│   │   ├── query.go            # Synchronous RAG
│   │   ├── rag_pipeline.go     # Shared RAG pipeline logic
│   │   ├── enqueue_query.go    # Async job enqueue
│   │   └── get_job.go          # Job status retrieval
│   │
│   └── middleware/
│       ├── logger.go
│       ├── metrics.go
│       └── rate_limit.go
│
└── .github/workflows/
    ├── ci.yml
    └── fly-deploy.yml
```
---

## 🧠 Architecture Overview

```text
+-----------------------+
|      HTTP Client      |
+-----------+-----------+
            |
            v
    +-------+--------+
    |     Fiber API  |
    +-------+--------+
            |
    +-------+--------+
    |                |
    v                v
+----+----+     +----+-----+
| Handlers |     | Middleware|
| notes.go |     | logger.go |
| query.go |     | metrics.go|
+----+----+     +-----------+
        |
        v
+------+------------------------------+
|               AI Layer              |
| embeddings.go   openai.go           |
| responder.go    mock/real toggle    |
+-------------------------------------+
        |
        v
+------+------------------------------+
|      Postgres 16 + pgvector         |
|  notes + note_embeddings tables     |
+-------------------------------------+
```

---

## 🔍 RAG Pipeline

```text
User Query
    |
    v
Generate Query Embedding (mock or real)
    |
    v
pgvector similarity search (<->)
    |
    v
Top-K Relevant Notes
    |
    +-----------------------------+
    | USE_MOCK_LLM=true  → Mock   |
    | USE_MOCK_LLM=false → Real   |
    +-----------------------------+
                    |
                    v
             Final AI Answer
```

The RAG pipeline is implemented once and reused by both synchronous HTTP handlers and the background worker.

---

## 🛠️ Running the Project

### 1. Clone

    git clone https://github.com/ai-backend-course/notes-memory-core-rag.git
    cd notes-memory-core-rag

### 2. Create your .env

    cp .env.example .env

Default mode:
- mock embeddings
- mock LLM
- no API key needed

### 3. Run with Docker

    docker-compose up --build

API available at:

    http://localhost:8081

---


## 📡 Endpoints

### GET /health
Health check.

### GET /notes
Return all notes.

### POST /notes

    {
      "title": "My Note",
      "content": "This is a test note."
    }

Creates:
- note record
- embedding (mock or real)

### POST /search

    {
      "query": "memory tips"
    }

Semantic vector search.

### POST /query (Synchronous RAG)

    {
      "query": "summarize my notes"
    }

Full RAG pipeline:
- semantic search
- top-k notes
- AI answer (mock or real)
- Context-aware execution with strict end-to-end timeouts
- Intended for demos, CLI usage, and lightweight UI interactions

This endpoint is always available, even when background infrastructure is not present.

###  POST /jobs/query & GET /jobs/:id Asynchronous RAG Jobs (Optional / Local & Extended Deployments)
- Enqueues RAG work into Redis
- Processes jobs with a background worker with retries and backoff
- Designed for long-running or high-latency AI tasks

If Redis is unavailable (e.g., API-only deployments), these endpoints return a clear `503 Service Unavailable` response instead of failing.

### GET /metrics

    {
      "total_requests": 12,
      "total_errors": 0,
      "avg_latency_ms": 1.7
    }

---

## 🤖 Using Real OpenAI 

Inside `.env`:

    USE_MOCK_EMBEDDINGS=false
    USE_MOCK_LLM=false
    OPENAI_API_KEY=your_key_here

This switches pipeline to:
- SmallEmbedding3 for embeddings
- GPT-4o Mini for generation

---

## 🧪 curl Examples

### Create note

    curl -X POST http://localhost:8081/notes \
      -H "Content-Type: application/json" \
      -d '{"title":"Test","content":"This is a demo note."}'

### Search

    curl -X POST http://localhost:8081/search \
      -H "Content-Type: application/json" \
      -d '{"query":"demo"}'

### RAG (sync)

    curl -X POST http://localhost:8081/query \
      -H "Content-Type: application/json" \
      -d '{"query":"summarize my notes"}'

### RAG (async) & Retrieve Status by ID

    curl -X POST http://localhost:8081/jobs/query \
      -H "Content-Type: application/json" \
      -d '{"query":"summarize my notes"}'

    curl http://localhost:8081/jobs/:id

---

## Production Deployment Behavior (Fly.io)

This service is deployed to Fly.io in an **API-only mode**:

- The synchronous RAG endpoint (`/query`) is always available
- Background job endpoints (`/jobs/*`) are enabled only when Redis is present
- Redis is treated as an optional dependency
- When Redis is unavailable, async endpoints return a clear `503 Service Unavailable`

This design demonstrates **graceful degradation** and allows the core API to remain stable even when optional infrastructure is absent.

---

## Reliability & Safety Guarantees

- All AI calls propagate `context.Context`
- Strict timeouts are enforced across the full RAG pipeline
- Long-running or blocked AI calls cannot stall the API
- Async jobs include retries with exponential backoff
- Optional infrastructure failures never crash the service

---

## 🧩 Tech Stack

| Component     | Technology       |
|---------------|------------------|
| Language      | Go 1.23          |
| Framework     | Fiber v2         |
| Database      | Postgres 16      |
| Vector Search | pgvector         |
| Embeddings    | SmallEmbedding3  |
| LLM           | GPT-4o Mini      |
| Containers    | Docker Compose   |
| Logging       | zerolog          |

---

## ⭐ Final Notes

This repo is part of a four-project AI Backend Portfolio:

1. notes-memory-core — template backend  
2. **notes-memory-core-rag — flagship RAG system**  
3. AI Summary Microservice  
4. Embedding Worker Microservice  
5. Portfolio Website 

This repository:

- Runs without OpenAI keys
- Fully supports real OpenAI
- Uses enterprise Go patterns
- Provides semantic search + RAG
- Is ready for employer review
- CI/CD is handled via GitHub Actions, automatically building and deploying to Fly.io with zero-downtime machine replacement.

