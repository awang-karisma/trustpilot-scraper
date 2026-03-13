#!/bin/sh

# Start cron daemon
crond -l 2

# Run scraper once on startup
echo "Running initial scrape..."
/app/scraper

# Keep container alive by tailing logs
touch /var/log/cron.log
tail -f /var/log/cron.log
