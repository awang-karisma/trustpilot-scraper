#!/bin/sh

# Ensure required directories exist
mkdir -p /app/data /app/templates

# Copy default templates if they don't exist in the bind mount
for f in discord.json slack.json; do
  if [ ! -f "/app/templates/$f" ]; then
    echo "Copying default template $f to /app/templates"
    cp "/app/defaults/$f" "/app/templates/"
  fi
done

# Start API server if enabled
if [ "${API_ENABLED:-true}" = "true" ]; then
  echo "Starting API server on ${API_HOST:-0.0.0.0}:${API_PORT:-8080}"
  /app/service &
  SERVICE_PID=$!
else
  echo "API server disabled"
  SERVICE_PID=""
fi

# Run scraper once on startup
echo "Running initial scrape..."
/app/scraper

# Wait for service to keep container alive
if [ -n "$SERVICE_PID" ]; then
  wait $SERVICE_PID
else
  echo "No service running, exiting..."
  exit 1
fi
