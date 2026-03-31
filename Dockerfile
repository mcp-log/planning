# syntax=docker/dockerfile:1

# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

WORKDIR /build

# Copy module files for dependency caching
COPY go.mod go.sum ./
COPY pkg/go.mod pkg/go.sum ./pkg/

# Download dependencies
RUN go mod download
RUN cd pkg && go mod download

# Copy source code
COPY pkg/ ./pkg/
COPY internal/ ./internal/
COPY cmd/ ./cmd/
COPY migrations/ ./migrations/
COPY api/ ./api/

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/planning \
    ./cmd/planning

# Runtime stage
FROM alpine:3.19

# Install CA certificates for HTTPS and timezone data
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 app && \
    adduser -D -u 1000 -G app app

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/planning /app/planning
COPY --from=builder /build/migrations /app/migrations

# Change ownership
RUN chown -R app:app /app

# Switch to non-root user
USER app

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Set entrypoint
ENTRYPOINT ["/app/planning"]
