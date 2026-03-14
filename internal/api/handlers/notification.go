package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"

	"github.com/awang-karisma/trustpilot-scraper/internal/api/dto"
	"github.com/awang-karisma/trustpilot-scraper/internal/config"
	"github.com/awang-karisma/trustpilot-scraper/internal/database"
	"github.com/awang-karisma/trustpilot-scraper/internal/scheduler"
)

// NotificationHandler handles notification channel requests
type NotificationHandler struct {
	db        *gorm.DB
	scheduler *scheduler.Scheduler
	config    *config.ServiceConfig
}

// NewNotificationHandler creates a new NotificationHandler
func NewNotificationHandler(db *gorm.DB, sched *scheduler.Scheduler, cfg *config.ServiceConfig) *NotificationHandler {
	return &NotificationHandler{
		db:        db,
		scheduler: sched,
		config:    cfg,
	}
}

// List returns all notification channels
// @Summary List notification channels
// @Description Get a list of all notification channels
// @Tags notifications
// @Produce json
// @Param enabled query bool false "Filter by enabled status"
// @Param website_id query string false "Filter by website ID"
// @Param limit query int false "Number of results" default(50)
// @Success 200 {object} dto.NotificationListResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/notifications [get]
func (h *NotificationHandler) List(c fiber.Ctx) error {
	enabled := c.Query("enabled") == "true"
	websiteID := c.Query("website_id")
	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = min(max(parsed, 1), 100)
		}
	}

	query := h.db.Model(&database.NotificationChannel{}).Preload("Website").Preload("Template")

	if enabled {
		query = query.Where("enabled = ?", true)
	}
	if websiteID != "" {
		query = query.Where("website_id = ?", websiteID)
	}

	var total int64
	query.Count(&total)

	var channels []database.NotificationChannel
	result := query.Order("created_at DESC").Limit(limit).Find(&channels)
	if result.Error != nil {
		return c.Status(500).JSON(dto.ErrorResponse{Error: result.Error.Error()})
	}

	return c.JSON(dto.ToNotificationListResponse(channels, total))
}

// Get returns a single notification channel by ID
// @Summary Get a notification channel
// @Description Get a notification channel by ID
// @Tags notifications
// @Produce json
// @Param id path string true "Notification Channel ID"
// @Success 200 {object} dto.NotificationResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/notifications/{id} [get]
func (h *NotificationHandler) Get(c fiber.Ctx) error {
	id := c.Params("id")

	var channel database.NotificationChannel
	result := h.db.Preload("Website").Preload("Template").First(&channel, "id = ?", id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(dto.ErrorResponse{Error: "notification channel not found"})
		}
		return c.Status(500).JSON(dto.ErrorResponse{Error: result.Error.Error()})
	}

	return c.JSON(dto.ToNotificationResponse(channel))
}

// Create creates a new notification channel
// @Summary Create a new notification channel
// @Description Register a new notification channel
// @Tags notifications
// @Accept json
// @Produce json
// @Param request body dto.CreateNotificationRequest true "Notification channel details"
// @Success 201 {object} dto.NotificationResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/notifications [post]
func (h *NotificationHandler) Create(c fiber.Ctx) error {
	var req dto.CreateNotificationRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(400).JSON(dto.ErrorResponse{Error: "invalid request body"})
	}

	// Validate website exists
	var website database.Website
	if result := h.db.First(&website, "id = ?", req.WebsiteID); result.Error != nil {
		return c.Status(404).JSON(dto.ErrorResponse{Error: "website not found"})
	}

	// Validate template exists
	var template database.Template
	if result := h.db.First(&template, "id = ?", req.TemplateID); result.Error != nil {
		return c.Status(404).JSON(dto.ErrorResponse{Error: "template not found"})
	}

	channel := database.NotificationChannel{
		Name:       req.Name,
		Schedule:   req.Schedule,
		WebsiteID:  req.WebsiteID,
		TemplateID: req.TemplateID,
		WebhookURL: req.WebhookURL,
		Enabled:    req.Enabled,
	}

	result := h.db.Create(&channel)
	if result.Error != nil {
		return c.Status(500).JSON(dto.ErrorResponse{Error: result.Error.Error()})
	}

	// Schedule the channel if enabled
	if h.scheduler != nil && channel.Enabled {
		h.scheduler.ScheduleNotificationChannel(channel)
	}

	return c.Status(201).JSON(dto.ToNotificationResponse(channel))
}

// Update updates an existing notification channel
// @Summary Update a notification channel
// @Description Update notification channel details by ID
// @Tags notifications
// @Accept json
// @Produce json
// @Param id path string true "Notification Channel ID"
// @Param request body dto.UpdateNotificationRequest true "Notification channel details"
// @Success 200 {object} dto.NotificationResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/notifications/{id} [put]
func (h *NotificationHandler) Update(c fiber.Ctx) error {
	id := c.Params("id")

	var channel database.NotificationChannel
	result := h.db.Preload("Website").Preload("Template").First(&channel, "id = ?", id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(dto.ErrorResponse{Error: "notification channel not found"})
		}
		return c.Status(500).JSON(dto.ErrorResponse{Error: result.Error.Error()})
	}

	var req dto.UpdateNotificationRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(400).JSON(dto.ErrorResponse{Error: "invalid request body"})
	}

	// Apply updates
	if req.Name != nil {
		channel.Name = *req.Name
	}
	if req.Schedule != nil {
		channel.Schedule = *req.Schedule
	}
	if req.WebsiteID != nil {
		channel.WebsiteID = *req.WebsiteID
	}
	if req.TemplateID != nil {
		channel.TemplateID = *req.TemplateID
	}
	if req.WebhookURL != nil {
		channel.WebhookURL = *req.WebhookURL
	}
	if req.Enabled != nil {
		channel.Enabled = *req.Enabled
	}

	result = h.db.Save(&channel)
	if result.Error != nil {
		return c.Status(500).JSON(dto.ErrorResponse{Error: result.Error.Error()})
	}

	// Update scheduler
	if h.scheduler != nil {
		if channel.Enabled {
			h.scheduler.ScheduleNotificationChannel(channel)
		} else {
			h.scheduler.UnscheduleNotificationChannel(channel.ID)
		}
	}

	return c.JSON(dto.ToNotificationResponse(channel))
}

// Delete deletes a notification channel
// @Summary Delete a notification channel
// @Description Delete a notification channel by ID
// @Tags notifications
// @Produce json
// @Param id path string true "Notification Channel ID"
// @Success 200 {object} dto.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/notifications/{id} [delete]
func (h *NotificationHandler) Delete(c fiber.Ctx) error {
	id := c.Params("id")

	// Unschedule before deleting
	if h.scheduler != nil {
		h.scheduler.UnscheduleNotificationChannel(id)
	}

	result := h.db.Delete(&database.NotificationChannel{}, "id = ?", id)
	if result.Error != nil {
		return c.Status(500).JSON(dto.ErrorResponse{Error: result.Error.Error()})
	}

	if result.RowsAffected == 0 {
		return c.Status(404).JSON(dto.ErrorResponse{Error: "notification channel not found"})
	}

	return c.JSON(dto.SuccessResponse{Success: true, Message: "notification channel deleted"})
}

// Trigger triggers an immediate notification
// @Summary Trigger a notification
// @Description Manually trigger an immediate notification for a channel
// @Tags notifications
// @Produce json
// @Param id path string true "Notification Channel ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/notifications/{id}/trigger [post]
func (h *NotificationHandler) Trigger(c fiber.Ctx) error {
	id := c.Params("id")

	var channel database.NotificationChannel
	result := h.db.Preload("Website").Preload("Template").First(&channel, "id = ?", id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(dto.ErrorResponse{Error: "notification channel not found"})
		}
		return c.Status(500).JSON(dto.ErrorResponse{Error: result.Error.Error()})
	}

	// Trigger immediate notification via scheduler
	if h.scheduler != nil {
		if err := h.scheduler.TriggerNotificationImmediate(channel.ID); err != nil {
			return c.Status(500).JSON(dto.ErrorResponse{
				Error: "Failed to queue notification job: " + err.Error(),
			})
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "notification job queued",
		"channel": dto.ToNotificationResponse(channel),
	})
}

// ListJobs returns notification jobs for a channel
// @Summary List notification jobs
// @Description Get notification jobs for a specific channel
// @Tags notifications
// @Produce json
// @Param id path string true "Notification Channel ID"
// @Param status query string false "Filter by status (pending, sent, failed)"
// @Param limit query int false "Number of results" default(50)
// @Success 200 {object} dto.NotificationJobListResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/notifications/{id}/jobs [get]
func (h *NotificationHandler) ListJobs(c fiber.Ctx) error {
	id := c.Params("id")
	status := c.Query("status")
	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = min(max(parsed, 1), 100)
		}
	}

	query := h.db.Model(&database.NotificationJob{}).Where("channel_id = ?", id)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var jobs []database.NotificationJob
	result := query.Order("created_at DESC").Limit(limit).Find(&jobs)
	if result.Error != nil {
		return c.Status(500).JSON(dto.ErrorResponse{Error: result.Error.Error()})
	}

	return c.JSON(dto.ToNotificationJobListResponse(jobs, total))
}
