# ---------------------------------------------------
# Build Stage
# ---------------------------------------------------
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install Git (needed for go mod)
RUN apk add --no-cache git

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN go build -o notes-memory-core-rag .

# ---------------------------------------------------
# Run Stage
# ---------------------------------------------------
FROM alpine:latest

WORKDIR /app

# Add CA certificates so HTTPS calls (OpenAI) work
RUN apk add --no-cache ca-certificates

# Copy binary from builder stage
COPY --from=builder /app/notes-memory-core-rag .

# Expose API port
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- http://localhost:8080/health || exit 1

# Run the server
CMD ["./notes-memory-core-rag"]
