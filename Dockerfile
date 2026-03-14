# Build stage
FROM golang:1.25.7-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build both scraper and service binaries
RUN go build -o scraper ./cmd/scraper
RUN go build -o service ./cmd/service

# Final stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create required directories
RUN mkdir -p /app/data /app/templates /app/defaults

# Copy binaries
COPY --from=builder /app/scraper .
COPY --from=builder /app/service .

# Copy default templates to a separate location for fallback
COPY --from=builder /app/templates /app/defaults

# Copy Swagger documentation
COPY --from=builder /app/docs /app/docs

# Copy entrypoint script
COPY --from=builder /app/scripts/entrypoint.sh /app/entrypoint.sh

# Make entrypoint executable
RUN chmod +x /app/entrypoint.sh

# Set environment variables for defaults
ENV DATABASE_URL=postgres://postgres:postgres@db:5432/trustpilot?sslmode=disable
ENV WEBHOOK_TEMPLATE_PATH=/app/templates/discord.json
ENV API_ENABLED=true
ENV API_PORT=8080
ENV API_HOST=0.0.0.0

# Expose API port
EXPOSE 8080

ENTRYPOINT ["/app/entrypoint.sh"]
