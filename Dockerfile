# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies for CGO (required for SQLite)
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with CGO enabled (required for sqlite3)
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-linkmode external -extldflags "-static"' -o alfred .

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/alfred .

# Create data directory for persistent storage
RUN mkdir -p /data

# Expose port
EXPOSE 8080

# Run the application
CMD ["./alfred"]
