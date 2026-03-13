package database

import (
	"time"

	"gorm.io/gorm"
)

// Website represents a target website to scrape
type Website struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Name        string     `gorm:"uniqueIndex;not null" json:"name"`     // e.g., "example.com"
	BaseURL     string     `gorm:"uniqueIndex;not null" json:"base_url"` // e.g., "https://example.com"
	Schedule    string     `gorm:"not null" json:"schedule"`             // Cron expression, e.g., "0 * * * *" (hourly)
	Enabled     bool       `gorm:"default:true" json:"enabled"`          // Whether scraping is enabled
	LastScraped *time.Time `json:"last_scraped,omitempty"`               // Last successful scrape time
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Review represents a single review scraped from Trustpilot
type Review struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	ReviewID  string         `gorm:"uniqueIndex" json:"review_id"`     // Trustpilot review ID
	WebsiteID uint           `gorm:"index;not null" json:"website_id"` // FK to Website
	Website   Website        `gorm:"foreignKey:WebsiteID" json:"website,omitempty"`
	Reviewer  string         `json:"reviewer"`
	Title     string         `json:"title"`
	Content   string         `json:"content"`
	Rating    int            `json:"rating"` // 1-5 stars
	Date      time.Time      `json:"date"`   // Review date on Trustpilot
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// WebsiteRating represents a snapshot of overall rating for a website
type WebsiteRating struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	WebsiteID uint      `gorm:"index;not null" json:"website_id"` // FK to Website
	Website   Website   `gorm:"foreignKey:WebsiteID" json:"website,omitempty"`
	Rating    float64   `json:"rating"`     // Overall rating (e.g., 3.5)
	Count     int       `json:"count"`      // Total review count
	CreatedAt time.Time `json:"created_at"` // When this snapshot was taken
}

// ScrapeJob represents a scraping job execution
type ScrapeJob struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	WebsiteID    uint       `gorm:"index;not null" json:"website_id"` // FK to Website
	Website      Website    `gorm:"foreignKey:WebsiteID" json:"website,omitempty"`
	Status       string     `gorm:"index;not null" json:"status"` // pending, running, completed, failed
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	Error        string     `json:"error,omitempty"`                // Error message if failed
	ReviewsFound int        `gorm:"default:0" json:"reviews_found"` // Number of reviews found
	CreatedAt    time.Time  `json:"created_at"`
}

// Job status constants
const (
	JobStatusPending   = "pending"
	JobStatusRunning   = "running"
	JobStatusCompleted = "completed"
	JobStatusFailed    = "failed"
)

// Summary is kept for backward compatibility with existing code
// Deprecated: Use WebsiteRating instead
type Summary struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Rating    float64   `json:"rating"`
	Count     int       `json:"count"`
	CreatedAt time.Time `json:"created_at"`
}
