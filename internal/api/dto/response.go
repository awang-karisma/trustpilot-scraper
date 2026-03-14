package dto

import (
	"time"

	"github.com/awang-karisma/trustpilot-scraper/internal/database"
)

// Common responses

type ErrorResponse struct {
	Error string `json:"error"`
}

type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

type Pagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// Website responses

type WebsiteResponse struct {
	ID          uint       `json:"id"`
	Name        string     `json:"name"`
	BaseURL     string     `json:"base_url"`
	Schedule    string     `json:"schedule"`
	Enabled     bool       `json:"enabled"`
	MaxPages    int        `json:"max_pages"`
	LastScraped *time.Time `json:"last_scraped,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type WebsiteListResponse struct {
	Data  []WebsiteResponse `json:"data"`
	Total int64             `json:"total"`
}

func ToWebsiteResponse(w database.Website) WebsiteResponse {
	return WebsiteResponse{
		ID:          w.ID,
		Name:        w.Name,
		BaseURL:     w.BaseURL,
		Schedule:    w.Schedule,
		Enabled:     w.Enabled,
		MaxPages:    w.MaxPages,
		LastScraped: w.LastScraped,
		CreatedAt:   w.CreatedAt,
		UpdatedAt:   w.UpdatedAt,
	}
}

func ToWebsiteListResponse(websites []database.Website, total int64) WebsiteListResponse {
	data := make([]WebsiteResponse, len(websites))
	for i, w := range websites {
		data[i] = ToWebsiteResponse(w)
	}
	return WebsiteListResponse{Data: data, Total: total}
}

// Review responses

type ReviewResponse struct {
	ID          uint      `json:"id"`
	ReviewID    string    `json:"review_id"`
	WebsiteID   uint      `json:"website_id"`
	WebsiteName string    `json:"website_name,omitempty"`
	Reviewer    string    `json:"reviewer"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Rating      int       `json:"rating"`
	Date        time.Time `json:"date"`
	CreatedAt   time.Time `json:"created_at"`
}

type ReviewListResponse struct {
	Data       []ReviewResponse `json:"data"`
	Pagination Pagination       `json:"pagination"`
}

func ToReviewResponse(r database.Review) ReviewResponse {
	response := ReviewResponse{
		ID:        r.ID,
		ReviewID:  r.ReviewID,
		WebsiteID: r.WebsiteID,
		Reviewer:  r.Reviewer,
		Title:     r.Title,
		Content:   r.Content,
		Rating:    r.Rating,
		Date:      r.Date,
		CreatedAt: r.CreatedAt,
	}
	if r.Website.ID != 0 {
		response.WebsiteName = r.Website.Name
	}
	return response
}

func ToReviewListResponse(reviews []database.Review, page, limit int, total int64) ReviewListResponse {
	data := make([]ReviewResponse, len(reviews))
	for i, r := range reviews {
		data[i] = ToReviewResponse(r)
	}
	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}
	return ReviewListResponse{
		Data: data,
		Pagination: Pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}
}

// Rating responses

type RatingSnapshot struct {
	Rating    float64   `json:"rating"`
	Count     int       `json:"count"`
	CreatedAt time.Time `json:"created_at"`
}

type RatingHistoryResponse struct {
	WebsiteID   uint             `json:"website_id"`
	WebsiteName string           `json:"website_name"`
	Current     *RatingSnapshot  `json:"current,omitempty"`
	History     []RatingSnapshot `json:"history"`
}

// Job responses

type JobResponse struct {
	ID           uint       `json:"id"`
	WebsiteID    uint       `json:"website_id"`
	WebsiteName  string     `json:"website_name,omitempty"`
	Status       string     `json:"status"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	Error        string     `json:"error,omitempty"`
	ReviewsFound int        `json:"reviews_found"`
	CreatedAt    time.Time  `json:"created_at"`
}

type JobListResponse struct {
	Data  []JobResponse `json:"data"`
	Total int64         `json:"total"`
}

func ToJobResponse(j database.ScrapeJob) JobResponse {
	response := JobResponse{
		ID:           j.ID,
		WebsiteID:    j.WebsiteID,
		Status:       j.Status,
		StartedAt:    j.StartedAt,
		CompletedAt:  j.CompletedAt,
		Error:        j.Error,
		ReviewsFound: j.ReviewsFound,
		CreatedAt:    j.CreatedAt,
	}
	if j.Website.ID != 0 {
		response.WebsiteName = j.Website.Name
	}
	return response
}

func ToJobListResponse(jobs []database.ScrapeJob, total int64) JobListResponse {
	data := make([]JobResponse, len(jobs))
	for i, j := range jobs {
		data[i] = ToJobResponse(j)
	}
	return JobListResponse{Data: data, Total: total}
}

// Template responses

type TemplateResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	FileName    string    `json:"file_name"`
	Description string    `json:"description,omitempty"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TemplateListResponse struct {
	Data  []TemplateResponse `json:"data"`
	Total int64              `json:"total"`
}

func ToTemplateResponse(t database.Template) TemplateResponse {
	return TemplateResponse{
		ID:          t.ID,
		Name:        t.Name,
		FileName:    t.FileName,
		Description: t.Description,
		Enabled:     t.Enabled,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

func ToTemplateListResponse(templates []database.Template, total int64) TemplateListResponse {
	data := make([]TemplateResponse, len(templates))
	for i, t := range templates {
		data[i] = ToTemplateResponse(t)
	}
	return TemplateListResponse{Data: data, Total: total}
}

// Notification responses

type NotificationResponse struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Schedule     string    `json:"schedule"`
	WebsiteID    string    `json:"website_id"`
	WebsiteName  string    `json:"website_name,omitempty"`
	TemplateID   string    `json:"template_id"`
	TemplateName string    `json:"template_name,omitempty"`
	WebhookURL   string    `json:"webhook_url"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type NotificationListResponse struct {
	Data  []NotificationResponse `json:"data"`
	Total int64                  `json:"total"`
}

func ToNotificationResponse(n database.NotificationChannel) NotificationResponse {
	response := NotificationResponse{
		ID:         n.ID,
		Name:       n.Name,
		Schedule:   n.Schedule,
		WebsiteID:  n.WebsiteID,
		TemplateID: n.TemplateID,
		WebhookURL: n.WebhookURL,
		Enabled:    n.Enabled,
		CreatedAt:  n.CreatedAt,
		UpdatedAt:  n.UpdatedAt,
	}
	if n.Website != nil {
		response.WebsiteName = n.Website.Name
	}
	if n.Template != nil {
		response.TemplateName = n.Template.Name
	}
	return response
}

func ToNotificationListResponse(channels []database.NotificationChannel, total int64) NotificationListResponse {
	data := make([]NotificationResponse, len(channels))
	for i, n := range channels {
		data[i] = ToNotificationResponse(n)
	}
	return NotificationListResponse{Data: data, Total: total}
}

// NotificationJob responses

type NotificationJobResponse struct {
	ID        string     `json:"id"`
	ChannelID string     `json:"channel_id"`
	Status    string     `json:"status"`
	SentAt    *time.Time `json:"sent_at,omitempty"`
	Error     string     `json:"error,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type NotificationJobListResponse struct {
	Data  []NotificationJobResponse `json:"data"`
	Total int64                     `json:"total"`
}

func ToNotificationJobResponse(j database.NotificationJob) NotificationJobResponse {
	return NotificationJobResponse{
		ID:        j.ID,
		ChannelID: j.ChannelID,
		Status:    j.Status,
		SentAt:    j.SentAt,
		Error:     j.Error,
		CreatedAt: j.CreatedAt,
	}
}

func ToNotificationJobListResponse(jobs []database.NotificationJob, total int64) NotificationJobListResponse {
	data := make([]NotificationJobResponse, len(jobs))
	for i, j := range jobs {
		data[i] = ToNotificationJobResponse(j)
	}
	return NotificationJobListResponse{Data: data, Total: total}
}

// Stats responses

type StatsResponse struct {
	Websites WebsiteStats `json:"websites"`
	Reviews  ReviewStats  `json:"reviews"`
	Jobs     JobStats     `json:"jobs"`
	Queue    QueueStats   `json:"queue"`
}

type WebsiteStats struct {
	Total    int64 `json:"total"`
	Enabled  int64 `json:"enabled"`
	Disabled int64 `json:"disabled"`
}

type ReviewStats struct {
	Total      int64 `json:"total"`
	BadReviews int64 `json:"bad_reviews"` // 1-2 star reviews
	ThisWeek   int64 `json:"this_week"`
}

type JobStats struct {
	Pending        int64 `json:"pending"`
	Running        int64 `json:"running"`
	CompletedToday int64 `json:"completed_today"`
	FailedToday    int64 `json:"failed_today"`
}

type QueueStats struct {
	Size          int `json:"size"`
	WorkersActive int `json:"workers_active"`
}
