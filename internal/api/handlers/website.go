package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"

	"github.com/awang-karisma/trustpilot-scraper/internal/api/dto"
	"github.com/awang-karisma/trustpilot-scraper/internal/database"
	"github.com/awang-karisma/trustpilot-scraper/internal/scheduler"
)

// WebsiteHandler handles website-related requests
type WebsiteHandler struct {
	db        *gorm.DB
	scheduler *scheduler.Scheduler
}

// NewWebsiteHandler creates a new WebsiteHandler
func NewWebsiteHandler(db *gorm.DB, sched *scheduler.Scheduler) *WebsiteHandler {
	return &WebsiteHandler{
		db:        db,
		scheduler: sched,
	}
}

// List returns all websites
// @Summary List all websites
// @Description Get a list of all registered websites to scrape
// @Tags websites
// @Produce json
// @Success 200 {object} dto.WebsiteListResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/websites [get]
func (h *WebsiteHandler) List(c fiber.Ctx) error {
	var websites []database.Website
	result := h.db.Find(&websites)
	if result.Error != nil {
		return c.Status(500).JSON(dto.ErrorResponse{Error: result.Error.Error()})
	}

	return c.JSON(dto.ToWebsiteListResponse(websites, result.RowsAffected))
}

// Get returns a single website by ID
// @Summary Get a website
// @Description Get a registered website by ID
// @Tags websites
// @Produce json
// @Param id path int true "Website ID"
// @Success 200 {object} dto.WebsiteResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Router /api/websites/{id} [get]
func (h *WebsiteHandler) Get(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(dto.ErrorResponse{Error: "invalid website id"})
	}

	var website database.Website
	result := h.db.First(&website, id)
	if result.Error != nil {
		return c.Status(404).JSON(dto.ErrorResponse{Error: "website not found"})
	}

	return c.JSON(dto.ToWebsiteResponse(website))
}

// Create creates a new website
// @Summary Create a new website
// @Description Register a new website for scraping
// @Tags websites
// @Accept json
// @Produce json
// @Param request body dto.CreateWebsiteRequest true "Website details"
// @Success 201 {object} dto.WebsiteResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/websites [post]
func (h *WebsiteHandler) Create(c fiber.Ctx) error {
	var req dto.CreateWebsiteRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(400).JSON(dto.ErrorResponse{Error: "invalid request body"})
	}

	website := database.Website{
		Name:     req.Name,
		BaseURL:  req.BaseURL,
		Schedule: req.Schedule,
		Enabled:  req.Enabled,
		MaxPages: req.MaxPages,
	}

	// Set default MaxPages if not provided
	if website.MaxPages <= 0 {
		website.MaxPages = 1
	}

	result := h.db.Create(&website)
	if result.Error != nil {
		return c.Status(500).JSON(dto.ErrorResponse{Error: result.Error.Error()})
	}

	// Schedule the website if enabled
	if h.scheduler != nil && website.Enabled {
		h.scheduler.ScheduleWebsite(website)
	}

	return c.Status(201).JSON(dto.ToWebsiteResponse(website))
}

// Update updates an existing website
// @Summary Update a website
// @Description Update registered website details by ID
// @Tags websites
// @Accept json
// @Produce json
// @Param id path int true "Website ID"
// @Param request body dto.UpdateWebsiteRequest true "Website details"
// @Success 200 {object} dto.WebsiteResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/websites/{id} [put]
func (h *WebsiteHandler) Update(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(dto.ErrorResponse{Error: "invalid website id"})
	}

	var website database.Website
	result := h.db.First(&website, id)
	if result.Error != nil {
		return c.Status(404).JSON(dto.ErrorResponse{Error: "website not found"})
	}

	var req dto.UpdateWebsiteRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(400).JSON(dto.ErrorResponse{Error: "invalid request body"})
	}

	// Apply updates
	if req.Name != nil {
		website.Name = *req.Name
	}
	if req.BaseURL != nil {
		website.BaseURL = *req.BaseURL
	}
	if req.Schedule != nil {
		website.Schedule = *req.Schedule
	}
	if req.Enabled != nil {
		website.Enabled = *req.Enabled
	}
	if req.MaxPages != nil {
		website.MaxPages = *req.MaxPages
		// Ensure MaxPages is at least 1
		if website.MaxPages <= 0 {
			website.MaxPages = 1
		}
	}

	result = h.db.Save(&website)
	if result.Error != nil {
		return c.Status(500).JSON(dto.ErrorResponse{Error: result.Error.Error()})
	}

	// Update scheduler
	if h.scheduler != nil {
		if website.Enabled {
			h.scheduler.ScheduleWebsite(website)
		} else {
			h.scheduler.UnscheduleWebsite(website.ID)
		}
	}

	return c.JSON(dto.ToWebsiteResponse(website))
}

// Delete deletes a website
// @Summary Delete a website
// @Description Delete a registered website by ID
// @Tags websites
// @Produce json
// @Param id path int true "Website ID"
// @Success 200 {object} dto.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/websites/{id} [delete]
func (h *WebsiteHandler) Delete(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(dto.ErrorResponse{Error: "invalid website id"})
	}

	// Unschedule before deleting
	if h.scheduler != nil {
		h.scheduler.UnscheduleWebsite(uint(id))
	}

	result := h.db.Delete(&database.Website{}, id)
	if result.Error != nil {
		return c.Status(500).JSON(dto.ErrorResponse{Error: result.Error.Error()})
	}

	if result.RowsAffected == 0 {
		return c.Status(404).JSON(dto.ErrorResponse{Error: "website not found"})
	}

	return c.JSON(dto.SuccessResponse{Success: true, Message: "website deleted"})
}

// TriggerScrape triggers an immediate scrape for a website
// @Summary Trigger a scrape
// @Description Manually trigger an immediate scraping job for a website
// @Tags websites
// @Produce json
// @Param id path int true "Website ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/websites/{id}/scrape [post]
func (h *WebsiteHandler) TriggerScrape(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(dto.ErrorResponse{Error: "invalid website id"})
	}

	var website database.Website
	result := h.db.First(&website, id)
	if result.Error != nil {
		return c.Status(404).JSON(dto.ErrorResponse{Error: "website not found"})
	}

	// Trigger immediate scrape via scheduler
	if h.scheduler != nil {
		if err := h.scheduler.TriggerImmediate(website.ID); err != nil {
			return c.Status(500).JSON(dto.ErrorResponse{
				Error: "Failed to queue scrape job: " + err.Error(),
			})
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "scrape job queued",
		"website": dto.ToWebsiteResponse(website),
	})
}
