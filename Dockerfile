# Stage 1: Build stage
FROM golang:alpine AS builder

# Install build tools required for CGO (gcc, musl-dev for go-sqlite3 C bindings)
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Download Go dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and web assets
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY web/ ./web/

# Compile the server binary with CGO enabled
RUN CGO_ENABLED=1 GOOS=linux go build -o server ./cmd/server

# Stage 2: Final minimal runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates sqlite-libs

# Create a non-root group and user
RUN addgroup -S forumgroup && adduser -S forumuser -G forumgroup

WORKDIR /app

# Copy compiled binary and web assets from the builder stage
COPY --from=builder /app/server ./server
COPY --from=builder /app/web ./web

# Set correct ownership for the non-root user
RUN chown -R forumuser:forumgroup /app

# Default environment configuration
ENV PORT=8080 \
    DB_PATH=/app/forum.db \
    SECURE_COOKIES=false

# Expose default HTTP port
EXPOSE 8080

# Run as non-root user
USER forumuser

# Start the forum server
CMD ["/app/server"]
