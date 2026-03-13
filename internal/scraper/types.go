package scraper

import (
	"github.com/awang-karisma/trustpilot-scraper/internal/database"
)

// ScrapeResult contains the unified scraping results
type ScrapeResult struct {
	Summary database.Summary
	Reviews []database.Review
}
