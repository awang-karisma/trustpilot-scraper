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

# Build the application
RUN go build -o scraper ./cmd/scraper

# Final stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create required directories
RUN mkdir -p /app/data /app/templates /app/defaults

# Copy binary
COPY --from=builder /app/scraper .

# Copy default templates to a separate location for fallback
COPY --from=builder /app/templates /app/defaults

# Setup Cron
RUN echo "0 * * * * /app/scraper >> /var/log/cron.log 2>&1" > /etc/crontabs/root

# Setup Entrypoint
RUN echo "#!/bin/sh" > /app/entrypoint.sh && \
    echo "mkdir -p /app/data /app/templates" >> /app/entrypoint.sh && \
    echo "# Copy default templates if they don't exist in the bind mount" >> /app/entrypoint.sh && \
    echo "for f in discord.json slack.json; do" >> /app/entrypoint.sh && \
    echo "  if [ ! -f \"/app/templates/\$f\" ]; then" >> /app/entrypoint.sh && \
    echo "    echo \"Copying default template \$f to /app/templates\"" >> /app/entrypoint.sh && \
    echo "    cp \"/app/defaults/\$f\" \"/app/templates/\"" >> /app/entrypoint.sh && \
    echo "  fi" >> /app/entrypoint.sh && \
    echo "done" >> /app/entrypoint.sh && \
    echo "touch /var/log/cron.log" >> /app/entrypoint.sh && \
    echo "echo 'Starting Cron...'" >> /app/entrypoint.sh && \
    echo "crond -L /var/log/cron.log" >> /app/entrypoint.sh && \
    echo "echo 'Running initial scrape...'" >> /app/entrypoint.sh && \
    echo "/app/scraper >> /var/log/cron.log 2>&1" >> /app/entrypoint.sh && \
    echo "tail -f /var/log/cron.log" >> /app/entrypoint.sh && \
    chmod +x /app/entrypoint.sh

# Initialize log file
RUN touch /var/log/cron.log

# Set environment variables for defaults
ENV DATABASE_URL=postgres://postgres:postgres@db:5432/trustpilot?sslmode=disable
ENV WEBHOOK_TEMPLATE_PATH=/app/templates/discord.json

ENTRYPOINT ["/app/entrypoint.sh"]
