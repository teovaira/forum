# Build stage
FROM golang:1.22-alpine AS builder

# Install build dependencies (GCC, musl-dev) for CGO SQLite
RUN apk add --no-cache build-base

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the Go application with CGO enabled
RUN CGO_ENABLED=1 GOOS=linux go build -o forum-server ./cmd/server

# Final stage
FROM alpine:3.20

# Install runtime dependencies for SQLite / C runtime
RUN apk add --no-cache ca-certificates sqlite-libs

# Create a non-root system user and group
RUN addgroup -S forum && adduser -S -G forum forum

# Create data directory for SQLite database volume
RUN mkdir -p /data && chown -R forum:forum /data

# Set working directory
WORKDIR /app

# Copy the pre-built binary
COPY --from=builder --chown=forum:forum /app/forum-server .

# Expose server port
EXPOSE 8080

# Switch to non-root user
USER forum

# Environment variables
ENV PORT=8080
ENV DB_PATH=/data/forum.db

# Mount database volume
VOLUME ["/data"]

# Run the server
CMD ["./forum-server"]
