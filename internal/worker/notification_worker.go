package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"gorm.io/gorm"

	"github.com/awang-karisma/trustpilot-scraper/internal/config"
	"github.com/awang-karisma/trustpilot-scraper/internal/database"
	"github.com/awang-karisma/trustpilot-scraper/internal/queue"
)

// NotificationWorker processes notification jobs from the queue
type NotificationWorker struct {
	id     int
	db     *gorm.DB
	q      queue.Queue
	cfg    *config.ServiceConfig
	logger *slog.Logger
}

// NewNotificationWorker creates a new notification worker
func NewNotificationWorker(id int, db *gorm.DB, q queue.Queue, cfg *config.ServiceConfig, logger *slog.Logger) *NotificationWorker {
	return &NotificationWorker{
		id:     id,
		db:     db,
		q:      q,
		cfg:    cfg,
		logger: logger.With("worker_id", id, "type", "notification"),
	}
}

// Start begins processing notification jobs from the queue
func (w *NotificationWorker) Start(ctx context.Context) {
	w.logger.Info("Notification worker started")

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Notification worker stopping")
			return
		default:
			if err := w.processNextJob(ctx); err != nil {
				w.logger.Error("Error processing notification job", "error", err)
			}
		}
	}
}

// processNextJob attempts to dequeue and process a notification job
func (w *NotificationWorker) processNextJob(ctx context.Context) error {
	// Dequeue with timeout context
	dequeueCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	job, err := w.q.Dequeue(dequeueCtx)
	if err != nil {
		// Context cancelled or queue stopped
		if err == context.Canceled || err == context.DeadlineExceeded {
			return nil
		}
		w.logger.Error("Failed to dequeue job", "error", err)
		return err
	}

	if job == nil {
		// No jobs available
		return nil
	}

	// Only process notification jobs
	if job.Type != "notification" {
		// Requeue non-notification jobs (they should be handled by scrape workers)
		w.logger.Warn("Received non-notification job, requeuing", "job_type", job.Type, "job_id", job.ID)
		if err := w.q.Nack(job.ID, true); err != nil {
			w.logger.Error("Failed to requeue job", "job_id", job.ID, "error", err)
		}
		return nil
	}

	w.processJob(ctx, job)
	return nil
}

// processJob processes a single notification job
func (w *NotificationWorker) processJob(ctx context.Context, job *queue.Job) {
	startTime := time.Now()
	w.logger.Info("Processing notification job", "job_id", job.ID, "channel_id", job.ChannelID, "attempt", job.Attempts+1)

	// Load channel with website and template
	var channel database.NotificationChannel
	if err := w.db.Preload("Website").Preload("Template").First(&channel, "id = ?", job.ChannelID).Error; err != nil {
		w.failJob(job, fmt.Sprintf("Failed to load channel: %v", err))
		return
	}

	// Get template path
	templatePath, err := w.getTemplatePath(channel.TemplateID)
	if err != nil {
		w.failJob(job, fmt.Sprintf("Failed to get template path: %v", err))
		return
	}

	// Fetch latest reviews for the website
	var reviews []database.Review
	if err := w.db.Where("website_id = ?", channel.WebsiteID).
		Order("date DESC").
		Limit(10).
		Find(&reviews).Error; err != nil {
		w.failJob(job, fmt.Sprintf("Failed to fetch reviews: %v", err))
		return
	}

	// Get latest rating
	var rating database.WebsiteRating
	if err := w.db.Where("website_id = ?", channel.WebsiteID).
		Order("created_at DESC").
		First(&rating).Error; err != nil {
		w.failJob(job, fmt.Sprintf("Failed to fetch rating: %v", err))
		return
	}

	// Send webhook for each review
	for _, review := range reviews {
		if err := w.sendWebhook(channel.WebhookURL, templatePath, channel.Website, review, rating); err != nil {
			w.failJob(job, fmt.Sprintf("Failed to send webhook: %v", err))
			return
		}
	}

	// Log successful job
	w.saveJobResult(job.ChannelID, database.NotificationJobStatusSent, nil)

	// Acknowledge job
	if err := w.q.Ack(job.ID); err != nil {
		w.logger.Error("Failed to acknowledge job", "job_id", job.ID, "error", err)
	}

	w.logger.Info("Notification job completed", "job_id", job.ID, "channel_id", job.ChannelID, "reviews_sent", len(reviews), "duration", time.Since(startTime))
}

// getTemplatePath resolves the template path with fallback
func (w *NotificationWorker) getTemplatePath(templateID string) (string, error) {
	// Try to find template in database
	var template database.Template
	result := w.db.Where("id = ? AND enabled = ?", templateID, true).First(&template)

	if result.Error == nil {
		// Construct full path: TEMPLATE_DIR + "/" + FileName
		fullPath := filepath.Join(w.cfg.TemplateDir, template.FileName)

		// Check if file exists
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath, nil
		}
		// File doesn't exist, log warning and continue to fallback
		w.logger.Warn("Template file not found, using fallback", "path", fullPath)
	}

	// Fallback to default template
	defaultPath := filepath.Join(w.cfg.TemplateDir, "discord.json")
	if _, err := os.Stat(defaultPath); err == nil {
		w.logger.Warn("Using default template", "path", defaultPath)
		return defaultPath, nil
	}

	// No template found
	return "", fmt.Errorf("template '%s' not found and no fallback available", templateID)
}

// sendWebhook sends a webhook notification
func (w *NotificationWorker) sendWebhook(webhookURL, templatePath string, website *database.Website, review database.Review, rating database.WebsiteRating) error {
	tmpl, err := template.New("webhook").Funcs(template.FuncMap{
		"json": func(v interface{}) string {
			b, _ := json.Marshal(v)
			s := string(b)
			if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
				return s[1 : len(s)-1]
			}
			return s
		},
	}).ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	tmplName := filepath.Base(templatePath)

	data := struct {
		Website     string
		Reviewer    string
		Title       string
		Content     string
		Rating      int
		TotalRating float64
		TotalCount  int
		Date        string
	}{
		Website:     website.Name,
		Reviewer:    review.Reviewer,
		Title:       review.Title,
		Content:     review.Content,
		Rating:      review.Rating,
		TotalRating: rating.Rating,
		TotalCount:  rating.Count,
		Date:        review.Date.Format("2006-01-02 15:04:05"),
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, tmplName, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	req, err := http.NewRequest("POST", webhookURL, &buf)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook returned non-success status: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// saveJobResult saves the notification job result to database
func (w *NotificationWorker) saveJobResult(channelID, status string, errorMsg *string) {
	job := database.NotificationJob{
		ChannelID: channelID,
		Status:    status,
	}

	if status == database.NotificationJobStatusSent {
		now := time.Now()
		job.SentAt = &now
	}

	if errorMsg != nil {
		job.Error = *errorMsg
	}

	if err := w.db.Create(&job).Error; err != nil {
		w.logger.Error("Failed to save notification job", "error", err)
	}
}

// failJob marks a job as failed and optionally requeues it
func (w *NotificationWorker) failJob(job *queue.Job, errMsg string) {
	w.saveJobResult(job.ChannelID, database.NotificationJobStatusFailed, &errMsg)

	// Nack with retry if attempts remaining
	retry := job.Attempts < job.MaxAttempts-1
	if err := w.q.Nack(job.ID, retry); err != nil {
		w.logger.Error("Failed to nack job", "job_id", job.ID, "error", err)
	}

	w.logger.Error("Notification job failed", "job_id", job.ID, "error", errMsg, "will_retry", retry)
}
