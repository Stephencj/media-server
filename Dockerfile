# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies for SQLite
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with CGO enabled for SQLite
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-linkmode external -extldflags "-static"' -o media-server ./cmd/server

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ffmpeg \
    ca-certificates \
    tzdata

# Create non-root user
RUN addgroup -g 1000 mediaserver && \
    adduser -u 1000 -G mediaserver -s /bin/sh -D mediaserver

# Create directories
RUN mkdir -p /data /media && \
    chown -R mediaserver:mediaserver /data

# Copy binary from builder
COPY --from=builder /app/media-server /usr/local/bin/media-server

# Copy web interface
COPY --from=builder /app/web /app/web

# Copy default config
COPY config.yaml /etc/media-server/config.yaml

WORKDIR /app

# Switch to non-root user
USER mediaserver

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Set default environment variables
ENV MEDIA_SERVER_HOST=0.0.0.0 \
    MEDIA_SERVER_PORT=8080 \
    MEDIA_SERVER_DB_PATH=/data/media-server.db \
    MEDIA_SERVER_ENV=production

CMD ["media-server"]
