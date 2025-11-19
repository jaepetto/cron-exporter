# Build stage
FROM golang:1.25.3-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with static linking (pure Go)
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags '-w -s -extldflags "-static"' \
    -o cronmetrics ./cmd/cronmetrics

# Final stage
FROM scratch

# Copy CA certificates for HTTPS requests
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary
COPY --from=builder /app/cronmetrics /cronmetrics

# Create data directory
VOLUME ["/data"]

# Expose ports
EXPOSE 8080 9090

# Set environment variables
ENV CRONMETRICS_DATABASE_PATH=/data/cronmetrics.db
ENV CRONMETRICS_SERVER_HOST=0.0.0.0
ENV CRONMETRICS_SERVER_PORT=8080
ENV CRONMETRICS_METRICS_PORT=9090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/cronmetrics", "version"] || exit 1

# Run as non-root user
USER 65534:65534

# Default command
ENTRYPOINT ["/cronmetrics"]
CMD ["serve"]
