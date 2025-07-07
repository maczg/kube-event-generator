# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o keg cmd/keg/main.go

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 keg && \
    adduser -D -u 1000 -G keg keg

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/keg /app/keg

# Copy default config
COPY --from=builder /app/config.yaml /app/config.yaml

# Create directories for scenarios and results
RUN mkdir -p /app/scenarios /app/results && \
    chown -R keg:keg /app

# Switch to non-root user
USER keg

# Expose health check port (if implemented)
EXPOSE 8080

# Set entrypoint
ENTRYPOINT ["/app/keg"]

# Default command
CMD ["--help"]
