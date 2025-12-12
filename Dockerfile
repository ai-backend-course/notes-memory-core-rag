# ---------- BUILD STAGE ----------
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install git (needed for go modules sometimes)
RUN apk add --no-cache git

# Copy go mod files first (cache optimization)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source
COPY . .

# Build API binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o api ./main.go

# Build WORKER binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o worker ./cmd/worker/main.go


# ---------- RUNTIME STAGE ----------
FROM alpine:latest

WORKDIR /app

# Copy binaries
COPY --from=builder /app/api ./api
COPY --from=builder /app/worker ./worker

# Expose API port
EXPOSE 8080

# Default command (API)
CMD ["./api"]
