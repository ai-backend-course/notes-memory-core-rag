# Notes Memory Core — RAG Extension

![Go Version](https://img.shields.io/badge/Go-1.23-blue)
![Fiber](https://img.shields.io/badge/Framework-Fiber%20v2-forestgreen)
![Postgres](https://img.shields.io/badge/Database-Postgres%2016-blue)
![License](https://img.shields.io/badge/License-MIT-green)


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

---

## 🚀 Features

### Core Backend
- CRUD Notes API
- Structured logging (zerolog)
- In-memory metrics at /metrics
- Automatic migrations
- Dockerized Postgres 16

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
│
├── main.go
├── go.mod
├── go.sum
├── Dockerfile
├── docker-compose.yml
├── .env.example
│
└── internal/
    ├── database/
    │   └── database.go
    ├── handlers/
    │   ├── notes.go
    │   └── query.go
    ├── ai/
    │   ├── embeddings.go
    │   ├── openai.go
    │   └── responder.go
    └── middleware/
        ├── logger.go
        └── metrics.go
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

### POST /query

    {
      "query": "summarize my notes"
    }

Full RAG pipeline:
- semantic search
- top-k notes
- AI answer (mock or real)

### GET /metrics

    {
      "total_requests": 12,
      "total_errors": 0,
      "avg_latency_ms": 1.7
    }

---

## 🤖 Using Real OpenAI (Optional)

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

### RAG

    curl -X POST http://localhost:8081/query \
      -H "Content-Type: application/json" \
      -d '{"query":"summarize my notes"}'

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
5. Portfolio Website (final week)

This repository:

- Runs without OpenAI keys
- Fully supports real OpenAI
- Uses enterprise Go patterns
- Provides semantic search + RAG
- Is ready for employer review
