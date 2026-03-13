package website

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/awang-karisma/trustpilot-scraper/internal/database"
)

// Manager handles website lifecycle and initialization
type Manager struct {
	db     *gorm.DB
	logger *slog.Logger
}

// NewManager creates a new website manager
func NewManager(db *gorm.DB, logger *slog.Logger) *Manager {
	return &Manager{
		db:     db,
		logger: logger,
	}
}

// InitializeFromEnv creates initial website entries from TRUSTPILOT_URL environment variable
// Format: comma-separated list of URLs or semicolon-separated for multiple URLs with schedules
// Examples:
//   - "https://www.trustpilot.com/review/example.com"
//   - "https://www.trustpilot.com/review/example.com|0 * * * *;https://www.trustpilot.com/review/test.com|*/30 * * * *"
//
// Default schedule: hourly (0 * * * *)
func (m *Manager) InitializeFromEnv(trustpilotURLs string, defaultSchedule string) error {
	if trustpilotURLs == "" {
		m.logger.Info("No TRUSTPILOT_URL provided, skipping website initialization")
		return nil
	}

	m.logger.Info("Initializing websites from environment", "urls", trustpilotURLs, "default_schedule", defaultSchedule)

	// Parse URLs - split by semicolon for multiple URLs
	urlEntries := strings.Split(trustpilotURLs, ";")

	for _, entry := range urlEntries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		var url, schedule string
		parts := strings.Split(entry, "|")

		if len(parts) == 1 {
			// Just URL, use default schedule
			url = strings.TrimSpace(parts[0])
			schedule = defaultSchedule
		} else if len(parts) == 2 {
			// URL with custom schedule
			url = strings.TrimSpace(parts[0])
			schedule = strings.TrimSpace(parts[1])
			if schedule == "" {
				schedule = defaultSchedule
			}
		} else {
			m.logger.Warn("Invalid URL entry format", "entry", entry)
			continue
		}

		// Extract domain name from URL
		name, err := m.extractDomainFromURL(url)
		if err != nil {
			m.logger.Warn("Failed to extract domain from URL", "url", url, "error", err)
			continue
		}

		// Build base URL (e.g., "https://example.com")
		baseURL, err := m.extractBaseURL(url)
		if err != nil {
			m.logger.Warn("Failed to extract base URL", "url", url, "error", err)
			continue
		}

		// Check if website already exists
		var existing database.Website
		result := m.db.Where("name = ? OR base_url = ?", name, baseURL).First(&existing)

		if result.Error == nil {
			// Website exists, skip
			m.logger.Debug("Website already exists", "name", name, "base_url", baseURL)
			continue
		}

		if result.Error != gorm.ErrRecordNotFound {
			m.logger.Warn("Failed to check existing website", "name", name, "error", result.Error)
			continue
		}

		// Create new website
		website := database.Website{
			Name:      name,
			BaseURL:   baseURL,
			Schedule:  schedule,
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := m.db.Create(&website).Error; err != nil {
			m.logger.Warn("Failed to create website", "name", name, "error", err)
			continue
		}

		m.logger.Info("Website created", "id", website.ID, "name", website.Name, "base_url", website.BaseURL, "schedule", website.Schedule)
	}

	return nil
}

// extractDomainFromURL extracts domain name from Trustpilot URL
// Example: "https://www.trustpilot.com/review/example.com" -> "example.com"
func (m *Manager) extractDomainFromURL(url string) (string, error) {
	// Remove trailing slashes
	url = strings.TrimSuffix(url, "/")

	// Split by /review/
	parts := strings.Split(url, "/review/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid trustpilot URL format")
	}

	// Get domain from second part
	domain := parts[1]
	if domain == "" {
		return "", fmt.Errorf("empty domain in URL")
	}

	return domain, nil
}

// extractBaseURL extracts base URL from Trustpilot URL
// Example: "https://www.trustpilot.com/review/example.com" -> "https://example.com"
func (m *Manager) extractBaseURL(trustpilotURL string) (string, error) {
	domain, err := m.extractDomainFromURL(trustpilotURL)
	if err != nil {
		return "", err
	}

	// Build base URL
	baseURL := "https://" + domain
	return baseURL, nil
}
