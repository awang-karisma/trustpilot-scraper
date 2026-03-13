package handlers

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"

	"github.com/awang-karisma/trustpilot-scraper/internal/api/dto"
	"github.com/awang-karisma/trustpilot-scraper/internal/database"
)

// RatingHandler handles rating-related requests
type RatingHandler struct {
	db *gorm.DB
}

// NewRatingHandler creates a new RatingHandler
func NewRatingHandler(db *gorm.DB) *RatingHandler {
	return &RatingHandler{db: db}
}

// GetHistory returns rating history for a website
// @Summary Get rating history
// @Description Get rating history for a website
// @Tags ratings
// @Produce json
// @Param id path int true "Website ID"
// @Param from query string false "Start date (YYYY-MM-DD)"
// @Param to query string false "End date (YYYY-MM-DD)"
// @Param limit query int false "Number of records" default(30)
// @Success 200 {object} dto.RatingHistoryResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/websites/{id}/ratings [get]
func (h *RatingHandler) GetHistory(c fiber.Ctx) error {
	websiteID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(dto.ErrorResponse{Error: "invalid website id"})
	}

	// Check website exists
	var website database.Website
	if result := h.db.First(&website, websiteID); result.Error != nil {
		return c.Status(404).JSON(dto.ErrorResponse{Error: "website not found"})
	}

	// Parse query params
	fromStr := c.Query("from")
	toStr := c.Query("to")
	limit := min(max(queryInt(c, "limit", 30), 1), 365)

	// Build query
	query := h.db.Model(&database.WebsiteRating{}).
		Where("website_id = ?", websiteID).
		Order("created_at DESC")

	// Apply date filters
	if fromStr != "" {
		if from, err := time.Parse("2006-01-02", fromStr); err == nil {
			query = query.Where("created_at >= ?", from)
		}
	}
	if toStr != "" {
		if to, err := time.Parse("2006-01-02", toStr); err == nil {
			query = query.Where("created_at <= ?", to.Add(24*time.Hour))
		}
	}

	// Get current rating (most recent)
	var current *database.WebsiteRating
	var currentRating database.WebsiteRating
	if result := query.Limit(1).First(&currentRating); result.Error == nil {
		current = &currentRating
	}

	// Get history
	var ratings []database.WebsiteRating
	query.Limit(limit).Find(&ratings)

	// Build response
	response := dto.RatingHistoryResponse{
		WebsiteID:   website.ID,
		WebsiteName: website.Name,
		History:     make([]dto.RatingSnapshot, len(ratings)),
	}

	if current != nil {
		response.Current = &dto.RatingSnapshot{
			Rating:    current.Rating,
			Count:     current.Count,
			CreatedAt: current.CreatedAt,
		}
	}

	for i, r := range ratings {
		response.History[i] = dto.RatingSnapshot{
			Rating:    r.Rating,
			Count:     r.Count,
			CreatedAt: r.CreatedAt,
		}
	}

	return c.JSON(response)
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
