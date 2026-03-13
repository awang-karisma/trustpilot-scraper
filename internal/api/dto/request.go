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

// Webhook requests

type WebhookTriggerRequest struct {
	ReviewID uint `json:"review_id,omitempty"` // Optional: specific review to send
}

type WebhookTestRequest struct {
	WebhookURL   string `json:"webhook_url" validate:"required,url"`
	WebhookType  string `json:"webhook_type"`
	TemplatePath string `json:"template_path"`
}

// Job requests

type JobListQuery struct {
	Status    string `query:"status" validate:"omitempty,oneof=pending running completed failed"`
	WebsiteID uint   `query:"website_id"`
	Limit     int    `query:"limit" validate:"min=1,max=100"`
}

// Rating requests

type RatingHistoryQuery struct {
	From  time.Time `query:"from"`
	To    time.Time `query:"to"`
	Limit int       `query:"limit" validate:"min=1,max=365"`
}
