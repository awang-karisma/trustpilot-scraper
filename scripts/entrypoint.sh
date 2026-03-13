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

# Start cron daemon
crond -l 2

# Run scraper once on startup
echo "Running initial scrape..."
/app/scraper

# Keep container alive
touch /var/log/cron.log

# If service is running, wait for it; otherwise just tail logs
if [ -n "$SERVICE_PID" ]; then
  wait $SERVICE_PID
else
  tail -f /var/log/cron.log
fi
