package handlers

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"

	"github.com/awang-karisma/trustpilot-scraper/internal/api/dto"
	"github.com/awang-karisma/trustpilot-scraper/internal/database"
)

// StatsHandler handles stats-related requests
type StatsHandler struct {
	db *gorm.DB
}

// NewStatsHandler creates a new StatsHandler
func NewStatsHandler(db *gorm.DB) *StatsHandler {
	return &StatsHandler{db: db}
}

// Get returns service statistics
// @Summary Get service statistics
// @Description Get overall service statistics including websites, reviews, jobs, and queue
// @Tags stats
// @Produce json
// @Success 200 {object} dto.StatsResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/stats [get]
func (h *StatsHandler) Get(c fiber.Ctx) error {
	// Website stats
	var totalWebsites, enabledWebsites int64
	h.db.Model(&database.Website{}).Count(&totalWebsites)
	h.db.Model(&database.Website{}).Where("enabled = ?", true).Count(&enabledWebsites)

	// Review stats
	var totalReviews, badReviews, thisWeekReviews int64
	h.db.Model(&database.Review{}).Count(&totalReviews)
	h.db.Model(&database.Review{}).Where("rating IN ?", []int{1, 2}).Count(&badReviews)

	oneWeekAgo := time.Now().AddDate(0, 0, -7)
	h.db.Model(&database.Review{}).Where("created_at >= ?", oneWeekAgo).Count(&thisWeekReviews)

	// Job stats
	var pendingJobs, runningJobs, completedToday, failedToday int64
	h.db.Model(&database.ScrapeJob{}).Where("status = ?", database.JobStatusPending).Count(&pendingJobs)
	h.db.Model(&database.ScrapeJob{}).Where("status = ?", database.JobStatusRunning).Count(&runningJobs)

	today := time.Now().Truncate(24 * time.Hour)
	h.db.Model(&database.ScrapeJob{}).
		Where("status = ? AND created_at >= ?", database.JobStatusCompleted, today).
		Count(&completedToday)
	h.db.Model(&database.ScrapeJob{}).
		Where("status = ? AND created_at >= ?", database.JobStatusFailed, today).
		Count(&failedToday)

	return c.JSON(dto.StatsResponse{
		Websites: dto.WebsiteStats{
			Total:    totalWebsites,
			Enabled:  enabledWebsites,
			Disabled: totalWebsites - enabledWebsites,
		},
		Reviews: dto.ReviewStats{
			Total:      totalReviews,
			BadReviews: badReviews,
			ThisWeek:   thisWeekReviews,
		},
		Jobs: dto.JobStats{
			Pending:        pendingJobs,
			Running:        runningJobs,
			CompletedToday: completedToday,
			FailedToday:    failedToday,
		},
		Queue: dto.QueueStats{
			Size:          int(pendingJobs),
			WorkersActive: int(runningJobs),
		},
	})
}
