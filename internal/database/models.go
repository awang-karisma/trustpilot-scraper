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
	MaxPages    int        `gorm:"default:1" json:"max_pages"`           // Number of pages to scrape (default: 1)
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

// Template represents a notification template file
type Template struct {
	ID          string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	Name        string         `gorm:"not null" json:"name"`                   // User-provided name (e.g., "Discord Alert")
	FileName    string         `gorm:"not null" json:"file_name"`              // Template filename (e.g., "discord.json")
	Description string         `gorm:"type:text" json:"description,omitempty"` // Optional description
	Enabled     bool           `gorm:"default:true;index" json:"enabled"`      // Whether template is active
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// NotificationChannel represents a notification channel with scheduling
type NotificationChannel struct {
	ID         string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	Name       string         `gorm:"not null;index" json:"name"`                            // Channel name
	Schedule   string         `gorm:"not null" json:"schedule"`                              // Cron expression
	WebsiteID  string         `gorm:"not null;index;foreignKey:WebsiteID" json:"website_id"` // FK to Website
	Website    *Website       `gorm:"foreignKey:WebsiteID" json:"website,omitempty"`
	TemplateID string         `gorm:"not null;index;foreignKey:TemplateID" json:"template_id"` // FK to Template
	Template   *Template      `gorm:"foreignKey:TemplateID" json:"template,omitempty"`
	WebhookURL string         `gorm:"not null" json:"webhook_url"`       // Webhook URL to send notifications
	Enabled    bool           `gorm:"default:true;index" json:"enabled"` // Whether channel is active
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// NotificationJob represents a notification job execution
type NotificationJob struct {
	ID        string               `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	ChannelID string               `gorm:"index;not null;foreignKey:ChannelID" json:"channel_id"` // FK to NotificationChannel
	Channel   *NotificationChannel `gorm:"foreignKey:ChannelID" json:"channel,omitempty"`
	Status    string               `gorm:"index;not null" json:"status"`     // pending, sent, failed
	SentAt    *time.Time           `json:"sent_at,omitempty"`                // When notification was sent
	Error     string               `gorm:"type:text" json:"error,omitempty"` // Error message if failed
	CreatedAt time.Time            `json:"created_at"`
}

// Notification job status constants
const (
	NotificationJobStatusPending = "pending"
	NotificationJobStatusSent    = "sent"
	NotificationJobStatusFailed  = "failed"
)
