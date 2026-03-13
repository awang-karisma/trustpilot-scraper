package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"

	"github.com/awang-karisma/trustpilot-scraper/internal/api/dto"
	"github.com/awang-karisma/trustpilot-scraper/internal/database"
)

// JobHandler handles job-related requests
type JobHandler struct {
	db *gorm.DB
}

// NewJobHandler creates a new JobHandler
func NewJobHandler(db *gorm.DB) *JobHandler {
	return &JobHandler{db: db}
}

// List returns jobs with filters
// @Summary List jobs
// @Description Get jobs with optional filters
// @Tags jobs
// @Produce json
// @Param status query string false "Filter by status (pending, running, completed, failed)"
// @Param website_id query int false "Filter by website ID"
// @Param limit query int false "Number of results" default(50)
// @Success 200 {object} dto.JobListResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/jobs [get]
func (h *JobHandler) List(c fiber.Ctx) error {
	// Parse query params
	status := c.Query("status")
	websiteID := queryInt(c, "website_id", 0)
	limit := min(max(queryInt(c, "limit", 50), 1), 100)

	// Validate status
	if status != "" && status != database.JobStatusPending &&
		status != database.JobStatusRunning &&
		status != database.JobStatusCompleted &&
		status != database.JobStatusFailed {
		return c.Status(400).JSON(dto.ErrorResponse{Error: "invalid status filter"})
	}

	// Build query
	query := h.db.Model(&database.ScrapeJob{}).Preload("Website")

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if websiteID > 0 {
		query = query.Where("website_id = ?", websiteID)
	}

	// Count total
	var total int64
	query.Count(&total)

	// Execute query
	var jobs []database.ScrapeJob
	result := query.Order("created_at DESC").Limit(limit).Find(&jobs)
	if result.Error != nil {
		return c.Status(500).JSON(dto.ErrorResponse{Error: result.Error.Error()})
	}

	return c.JSON(dto.ToJobListResponse(jobs, total))
}

// Get returns a single job by ID
// @Summary Get a job
// @Description Get a job by ID
// @Tags jobs
// @Produce json
// @Param id path int true "Job ID"
// @Success 200 {object} dto.JobResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/jobs/{id} [get]
func (h *JobHandler) Get(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(dto.ErrorResponse{Error: "invalid job id"})
	}

	var job database.ScrapeJob
	result := h.db.Preload("Website").First(&job, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(dto.ErrorResponse{Error: "job not found"})
		}
		return c.Status(500).JSON(dto.ErrorResponse{Error: result.Error.Error()})
	}

	return c.JSON(dto.ToJobResponse(job))
}
