package handlers

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"

	"github.com/awang-karisma/trustpilot-scraper/internal/api/dto"
	"github.com/awang-karisma/trustpilot-scraper/internal/database"
	"github.com/awang-karisma/trustpilot-scraper/internal/webhook"
)

// WebhookHandler handles webhook-related requests
type WebhookHandler struct {
	db      *gorm.DB
	webhook *webhook.WebhookService
}

// NewWebhookHandler creates a new WebhookHandler
func NewWebhookHandler(db *gorm.DB, webhookService *webhook.WebhookService) *WebhookHandler {
	return &WebhookHandler{
		db:      db,
		webhook: webhookService,
	}
}

// Trigger triggers a webhook for a specific website
// @Summary Trigger a webhook
// @Description Manually trigger a webhook for a specific website
// @Tags webhooks
// @Accept json
// @Produce json
// @Param id path int true "Website ID"
// @Param request body dto.WebhookTriggerRequest false "Webhook trigger options"
// @Success 200 {object} dto.WebhookTriggerResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/websites/{id}/webhook [post]
func (h *WebhookHandler) Trigger(c fiber.Ctx) error {
	websiteID := c.Params("id")

	// Get website
	var website database.Website
	if result := h.db.First(&website, websiteID); result.Error != nil {
		return c.Status(404).JSON(dto.ErrorResponse{Error: "website not found"})
	}

	// Parse optional request body
	var req dto.WebhookTriggerRequest
	_ = c.Bind().JSON(&req)

	// Get review if specified, or get worst review for this website
	var review database.Review
	if req.ReviewID > 0 {
		if result := h.db.First(&review, req.ReviewID); result.Error != nil {
			return c.Status(404).JSON(dto.ErrorResponse{Error: "review not found"})
		}
	} else {
		// Get worst review for this website
		result := h.db.Where("website_id = ?", websiteID).
			Order("rating ASC, date DESC").
			First(&review)
		if result.Error != nil {
			// No reviews found, use empty review
			review = database.Review{
				Reviewer: "N/A",
				Title:    "No reviews available",
				Content:  "No reviews have been scraped yet for this website.",
				Rating:   0,
				Date:     time.Now(),
			}
		}
	}

	// Get latest rating
	var rating database.WebsiteRating
	result := h.db.Where("website_id = ?", websiteID).
		Order("created_at DESC").
		First(&rating)
	if result.Error != nil {
		// No rating found, use zero values
		rating = database.WebsiteRating{
			Rating: 0,
			Count:  0,
		}
	}

	// Send webhook if configured
	if h.webhook != nil {
		// Convert WebsiteRating to Summary for compatibility with webhook
		summary := database.Summary{
			Rating: rating.Rating,
			Count:  rating.Count,
		}
		if err := h.webhook.Send(website.Name, review, summary); err != nil {
			return c.Status(500).JSON(dto.ErrorResponse{
				Error: "Failed to send webhook: " + err.Error(),
			})
		}
	}

	return c.JSON(dto.WebhookTriggerResponse{
		Success: true,
		Message: "Webhook sent successfully",
		SentAt:  time.Now(),
	})
}

// Test tests a webhook configuration
// @Summary Test a webhook
// @Description Test a webhook configuration with mock data
// @Tags webhooks
// @Accept json
// @Produce json
// @Param request body dto.WebhookTestRequest true "Webhook test configuration"
// @Success 200 {object} dto.WebhookTestResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/webhook/test [post]
func (h *WebhookHandler) Test(c fiber.Ctx) error {
	var req dto.WebhookTestRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(400).JSON(dto.ErrorResponse{Error: "invalid request body"})
	}

	// Create test webhook service
	testWebhook := webhook.NewWebhookService(req.WebhookURL, req.TemplatePath)

	// Create mock data
	mockReview := database.Review{
		Reviewer: "Test User",
		Title:    "Test Review",
		Content:  "This is a test webhook message",
		Rating:   1,
		Date:     time.Now(),
	}
	mockSummary := database.Summary{
		Rating: 3.5,
		Count:  100,
	}

	// Send test webhook
	if err := testWebhook.Send("test-website.com", mockReview, mockSummary); err != nil {
		return c.Status(500).JSON(dto.ErrorResponse{
			Error: "Webhook test failed: " + err.Error(),
		})
	}

	return c.JSON(dto.WebhookTestResponse{
		Success:        true,
		Message:        "Webhook test successful",
		ResponseStatus: 204,
	})
}
