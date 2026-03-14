package worker

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"

	"github.com/awang-karisma/trustpilot-scraper/internal/config"
	"github.com/awang-karisma/trustpilot-scraper/internal/database"
	"github.com/awang-karisma/trustpilot-scraper/internal/queue"
	"github.com/awang-karisma/trustpilot-scraper/internal/scraper"
)

// Worker processes scrape jobs from the queue
type Worker struct {
	id      int
	db      *gorm.DB
	queue   queue.Queue
	config  *config.ServiceConfig
	scraper *scraper.Scraper
	logger  *slog.Logger
}

// NewWorker creates a new worker
func NewWorker(id int, db *gorm.DB, q queue.Queue, cfg *config.ServiceConfig, logger *slog.Logger) *Worker {
	return &Worker{
		id:      id,
		db:      db,
		queue:   q,
		config:  cfg,
		scraper: scraper.NewScraper(""), // Initialize scraper here
		logger:  logger.With("worker_id", id),
	}
}

// Start begins processing jobs from the queue
func (w *Worker) Start(ctx context.Context) {
	w.logger.Info("Worker started")

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Worker stopping")
			return
		default:
			w.processNextJob(ctx)
		}
	}
}

// processNextJob attempts to dequeue and process a job
func (w *Worker) processNextJob(ctx context.Context) {
	// Dequeue with timeout context
	dequeueCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	job, err := w.queue.Dequeue(dequeueCtx)
	if err != nil {
		// Context cancelled or queue stopped
		if err == context.Canceled || err == context.DeadlineExceeded {
			return
		}
		w.logger.Error("Failed to dequeue job", "error", err)
		return
	}

	if job == nil {
		// No jobs available
		return
	}

	w.processJob(ctx, job)
}

// processJob processes a single scrape job
func (w *Worker) processJob(ctx context.Context, job *queue.Job) {
	startTime := time.Now()
	w.logger.Info("Processing job", "job_id", job.ID, "website_id", job.WebsiteID, "attempt", job.Attempts+1)

	// Create scrape job record
	scrapeJob := database.ScrapeJob{
		WebsiteID: job.WebsiteID,
		Status:    database.JobStatusRunning,
		StartedAt: &startTime,
	}
	w.db.Create(&scrapeJob)

	// Get website
	var website database.Website
	if err := w.db.First(&website, job.WebsiteID).Error; err != nil {
		w.failJob(&scrapeJob, job, fmt.Sprintf("Website not found: %v", err))
		return
	}

	// Scraper is already initialized in NewWorker, so no need to check here
	// if w.scraper == nil {
	// 	w.scraper = scraper.NewScraper("")
	// }

	// Determine max pages to scrape (default to 1 if not set)
	maxPages := website.MaxPages
	if maxPages <= 0 {
		maxPages = 1
	}

	// Create timeout context for scraping
	scrapeTimeout := time.Duration(w.config.ScrapeTimeoutSec) * time.Second
	scrapeCtx, cancel := context.WithTimeout(ctx, scrapeTimeout)
	defer cancel()

	// Scrape pages in parallel
	var allReviews []database.Review
	var summary database.Summary
	var err error

	if maxPages == 1 {
		// Single page scraping (backward compatible)
		badReviewsURL := w.buildBadReviewsURL(website.BaseURL, 1)
		result, scrapeErr := w.scraper.ScrapeWithContext(scrapeCtx, badReviewsURL)
		if scrapeErr != nil {
			w.failJob(&scrapeJob, job, fmt.Sprintf("Scraping failed: %v", scrapeErr))
			return
		}
		allReviews = result.Reviews
		summary = result.Summary
	} else {
		// Parallel multi-page scraping
		allReviews, summary, err = w.scrapePagesParallel(scrapeCtx, website.BaseURL, maxPages)
		if err != nil {
			w.failJob(&scrapeJob, job, fmt.Sprintf("Scraping failed: %v", err))
			return
		}
	}

	// Save results
	reviewsSaved, saveErr := w.saveResults(website, &scraper.ScrapeResult{
		Summary: summary,
		Reviews: allReviews,
	})
	if saveErr != nil {
		w.failJob(&scrapeJob, job, fmt.Sprintf("Failed to save results: %v", saveErr))
		return
	}

	// Update website last scraped
	now := time.Now()
	website.LastScraped = &now
	w.db.Save(&website)

	// Mark job as completed
	completedAt := time.Now()
	scrapeJob.Status = database.JobStatusCompleted
	scrapeJob.CompletedAt = &completedAt
	scrapeJob.ReviewsFound = reviewsSaved
	w.db.Save(&scrapeJob)

	// Acknowledge job
	if err := w.queue.Ack(job.ID); err != nil {
		w.logger.Error("Failed to ack job", "job_id", job.ID, "error", err)
	}

	w.logger.Info("Job completed", "job_id", job.ID, "website_id", job.WebsiteID, "reviews_saved", reviewsSaved, "duration", time.Since(startTime))
}

// buildBadReviewsURL constructs the URL for fetching bad reviews
func (w *Worker) buildBadReviewsURL(baseURL string, page int) string {
	// Extract website name from base URL
	// e.g., "https://example.com" -> "example.com"
	website := strings.TrimPrefix(baseURL, "https://")
	website = strings.TrimPrefix(website, "http://")
	website = strings.TrimSuffix(website, "/")

	// Build Trustpilot URL for bad reviews with pagination
	// Trustpilot uses page parameter for pagination
	if page <= 1 {
		return fmt.Sprintf("https://www.trustpilot.com/review/%s?stars=1&stars=2", website)
	}
	return fmt.Sprintf("https://www.trustpilot.com/review/%s?stars=1&stars=2&page=%d", website, page)
}

// saveResults saves scraped reviews and rating to database
func (w *Worker) saveResults(website database.Website, result *scraper.ScrapeResult) (int, error) {
	savedCount := 0

	// Save rating snapshot
	if result.Summary.Rating > 0 {
		rating := database.WebsiteRating{
			WebsiteID: website.ID,
			Rating:    result.Summary.Rating,
			Count:     result.Summary.Count,
		}
		if err := w.db.Create(&rating).Error; err != nil {
			w.logger.Error("Failed to save rating", "website_id", website.ID, "error", err)
		}
	}

	// Save reviews
	for _, review := range result.Reviews {
		// Check if review already exists
		var existing database.Review
		err := w.db.Where("review_id = ?", review.ReviewID).First(&existing).Error
		if err == nil {
			// Review exists, skip
			continue
		}

		// Create new review
		dbReview := database.Review{
			ReviewID:  review.ReviewID,
			WebsiteID: website.ID,
			Reviewer:  review.Reviewer,
			Title:     review.Title,
			Content:   review.Content,
			Rating:    review.Rating,
			Date:      review.Date,
		}

		if err := w.db.Create(&dbReview).Error; err != nil {
			w.logger.Error("Failed to save review", "review_id", review.ReviewID, "error", err)
			continue
		}
		savedCount++
	}

	return savedCount, nil
}

// scrapePagesParallel scrapes multiple pages in parallel using goroutines
func (w *Worker) scrapePagesParallel(ctx context.Context, baseURL string, maxPages int) ([]database.Review, database.Summary, error) {
	var allReviews []database.Review
	var mu sync.Mutex
	var summary database.Summary

	g, ctx := errgroup.WithContext(ctx)

	// Launch a goroutine for each page
	for page := 1; page <= maxPages; page++ {
		page := page // Capture for goroutine
		g.Go(func() error {
			// Each goroutine gets its own scraper/browser instance
			pageScraper := scraper.NewScraper("")
			pageURL := w.buildBadReviewsURL(baseURL, page)

			w.logger.Info("Scraping page", "page", page, "url", pageURL)

			result, err := pageScraper.ScrapeWithContext(ctx, pageURL)
			if err != nil {
				w.logger.Error("Failed to scrape page", "page", page, "error", err)
				return err
			}

			// Collect results safely
			mu.Lock()
			allReviews = append(allReviews, result.Reviews...)
			// Use summary from first page
			if page == 1 {
				summary = result.Summary
			}
			mu.Unlock()

			w.logger.Info("Scraped page successfully", "page", page, "reviews", len(result.Reviews))
			return nil
		})
	}

	// Wait for all goroutines to complete
	if err := g.Wait(); err != nil {
		return nil, database.Summary{}, err
	}

	w.logger.Info("Parallel scraping completed", "total_pages", maxPages, "total_reviews", len(allReviews))
	return allReviews, summary, nil
}

// failJob marks a job as failed and optionally requeues it
func (w *Worker) failJob(scrapeJob *database.ScrapeJob, job *queue.Job, errMsg string) {
	completedAt := time.Now()
	scrapeJob.Status = database.JobStatusFailed
	scrapeJob.CompletedAt = &completedAt
	scrapeJob.Error = errMsg
	w.db.Save(scrapeJob)

	// Nack with retry if attempts remaining
	retry := job.Attempts < job.MaxAttempts-1
	if err := w.queue.Nack(job.ID, retry); err != nil {
		w.logger.Error("Failed to nack job", "job_id", job.ID, "error", err)
	}

	w.logger.Error("Job failed", "job_id", job.ID, "error", errMsg, "will_retry", retry)
}
