package handlers

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"

	"github.com/awang-karisma/trustpilot-scraper/internal/api/dto"
	"github.com/awang-karisma/trustpilot-scraper/internal/database"
)

// ReviewHandler handles review-related requests
type ReviewHandler struct {
	db *gorm.DB
}

// NewReviewHandler creates a new ReviewHandler
func NewReviewHandler(db *gorm.DB) *ReviewHandler {
	return &ReviewHandler{db: db}
}

// queryInt parses a query parameter as int with default value
func queryInt(c fiber.Ctx, key string, defaultValue int) int {
	val := c.Query(key)
	if val == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return i
}

// List returns paginated reviews with filters
// @Summary List reviews
// @Description Get paginated reviews with optional filters
// @Tags reviews
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param website_id query int false "Filter by website ID"
// @Param rating query string false "Filter by rating (comma-separated, e.g., 1,2,3)"
// @Param sort query string false "Sort order (e.g., date desc, rating asc)" default(date desc)
// @Param search query string false "Search in title and content"
// @Success 200 {object} dto.ReviewListResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/reviews [get]
func (h *ReviewHandler) List(c fiber.Ctx) error {
	// Parse query params
	page := max(queryInt(c, "page", 1), 1)
	limit := min(max(queryInt(c, "limit", 20), 1), 100)
	websiteID := queryInt(c, "website_id", 0)
	rating := c.Query("rating")
	sort := c.Query("sort", "date desc")
	search := c.Query("search")

	// Build query
	query := h.db.Model(&database.Review{}).Preload("Website")

	// Apply filters
	if websiteID > 0 {
		query = query.Where("website_id = ?", websiteID)
	}

	if rating != "" {
		ratings := parseRatingFilter(rating)
		if len(ratings) > 0 {
			query = query.Where("rating IN ?", ratings)
		}
	}

	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where("title ILIKE ? OR content ILIKE ?", searchPattern, searchPattern)
	}

	// Count total
	var total int64
	query.Count(&total)

	// Apply sorting and pagination
	orderClause := parseSort(sort)
	query = query.Order(orderClause).Offset((page - 1) * limit).Limit(limit)

	// Execute query
	var reviews []database.Review
	result := query.Find(&reviews)
	if result.Error != nil {
		return c.Status(500).JSON(dto.ErrorResponse{Error: result.Error.Error()})
	}

	return c.JSON(dto.ToReviewListResponse(reviews, page, limit, total))
}

// Get returns a single review by ID
// @Summary Get a review
// @Description Get a review by ID
// @Tags reviews
// @Produce json
// @Param id path int true "Review ID"
// @Success 200 {object} dto.ReviewResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/reviews/{id} [get]
func (h *ReviewHandler) Get(c fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(dto.ErrorResponse{Error: "invalid review id"})
	}

	var review database.Review
	result := h.db.Preload("Website").First(&review, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(dto.ErrorResponse{Error: "review not found"})
		}
		return c.Status(500).JSON(dto.ErrorResponse{Error: result.Error.Error()})
	}

	return c.JSON(dto.ToReviewResponse(review))
}

// ListByWebsite returns reviews for a specific website
// @Summary List reviews by website
// @Description Get paginated reviews for a specific website
// @Tags reviews
// @Produce json
// @Param id path int true "Website ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param rating query string false "Filter by rating (comma-separated)"
// @Param sort query string false "Sort order" default(date desc)
// @Success 200 {object} dto.ReviewListResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/websites/{id}/reviews [get]
func (h *ReviewHandler) ListByWebsite(c fiber.Ctx) error {
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
	page := max(queryInt(c, "page", 1), 1)
	limit := min(max(queryInt(c, "limit", 20), 1), 100)
	rating := c.Query("rating")
	sort := c.Query("sort", "date desc")

	// Build query
	query := h.db.Model(&database.Review{}).Where("website_id = ?", websiteID)

	if rating != "" {
		ratings := parseRatingFilter(rating)
		if len(ratings) > 0 {
			query = query.Where("rating IN ?", ratings)
		}
	}

	// Count total
	var total int64
	query.Count(&total)

	// Apply sorting and pagination
	orderClause := parseSort(sort)
	query = query.Order(orderClause).Offset((page - 1) * limit).Limit(limit)

	// Execute query
	var reviews []database.Review
	result := query.Find(&reviews)
	if result.Error != nil {
		return c.Status(500).JSON(dto.ErrorResponse{Error: result.Error.Error()})
	}

	return c.JSON(dto.ToReviewListResponse(reviews, page, limit, total))
}

// parseRatingFilter parses rating filter string (e.g., "1,2" -> [1, 2])
func parseRatingFilter(rating string) []int {
	var ratings []int
	for _, r := range strings.Split(rating, ",") {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		var val int
		if _, err := strconv.Atoi(r); err == nil {
			val, _ = strconv.Atoi(r)
			if val >= 1 && val <= 5 {
				ratings = append(ratings, val)
			}
		}
	}
	return ratings
}

// parseSort parses sort string (e.g., "date desc" -> "date desc")
func parseSort(sort string) string {
	// Whitelist allowed sort columns
	allowedColumns := map[string]bool{
		"date":       true,
		"rating":     true,
		"created_at": true,
		"reviewer":   true,
	}

	parts := strings.Fields(sort)
	if len(parts) == 0 {
		return "date desc"
	}

	column := parts[0]
	order := "desc"
	if len(parts) > 1 && strings.ToLower(parts[1]) == "asc" {
		order = "asc"
	}

	if !allowedColumns[column] {
		return "date desc"
	}

	return column + " " + order
}
