package dto

import "time"

// Website requests

type CreateWebsiteRequest struct {
	Name     string `json:"name" validate:"required"`
	BaseURL  string `json:"base_url" validate:"required,url"`
	Schedule string `json:"schedule" validate:"omitempty"`
	Enabled  bool   `json:"enabled"`
	MaxPages int    `json:"max_pages" validate:"omitempty,min=1,max=50"`
}

type UpdateWebsiteRequest struct {
	Name     *string `json:"name,omitempty" validate:"omitempty,min=1"`
	BaseURL  *string `json:"base_url,omitempty" validate:"omitempty,url"`
	Schedule *string `json:"schedule,omitempty" validate:"omitempty"`
	Enabled  *bool   `json:"enabled,omitempty"`
	MaxPages *int    `json:"max_pages,omitempty" validate:"omitempty,min=1,max=50"`
}

// Review requests

type ReviewListQuery struct {
	Page      int    `query:"page" validate:"min=1"`
	Limit     int    `query:"limit" validate:"min=1,max=100"`
	WebsiteID uint   `query:"website_id"`
	Rating    string `query:"rating"` // e.g., "1,2" for bad reviews
	Sort      string `query:"sort"`   // e.g., "date desc", "rating asc"
	Search    string `query:"search"`
}

// Job requests

type JobListQuery struct {
	Status    string `query:"status" validate:"omitempty,oneof=pending running completed failed"`
	WebsiteID uint   `query:"website_id"`
	Limit     int    `query:"limit" validate:"min=1,max=100"`
}

// Template requests

type CreateTemplateRequest struct {
	Name        string `json:"name" validate:"required"`
	FileName    string `json:"file_name" validate:"required"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
}

type UpdateTemplateRequest struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=1"`
	FileName    *string `json:"file_name,omitempty" validate:"omitempty,min=1"`
	Description *string `json:"description,omitempty"`
	Enabled     *bool   `json:"enabled,omitempty"`
}

// Notification requests

type CreateNotificationRequest struct {
	Name       string `json:"name" validate:"required"`
	Schedule   string `json:"schedule" validate:"required"`
	WebsiteID  string `json:"website_id" validate:"required"`
	TemplateID string `json:"template_id" validate:"required"`
	WebhookURL string `json:"webhook_url" validate:"required,url"`
	Enabled    bool   `json:"enabled"`
}

type UpdateNotificationRequest struct {
	Name       *string `json:"name,omitempty" validate:"omitempty,min=1"`
	Schedule   *string `json:"schedule,omitempty" validate:"omitempty"`
	WebsiteID  *string `json:"website_id,omitempty" validate:"omitempty"`
	TemplateID *string `json:"template_id,omitempty" validate:"omitempty,min=1"`
	WebhookURL *string `json:"webhook_url,omitempty" validate:"omitempty,url"`
	Enabled    *bool   `json:"enabled,omitempty"`
}

// Rating requests

type RatingHistoryQuery struct {
	From  time.Time `query:"from"`
	To    time.Time `query:"to"`
	Limit int       `query:"limit" validate:"min=1,max=365"`
}
