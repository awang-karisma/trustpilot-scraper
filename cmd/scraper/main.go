package main

import (
	"github.com/awang-karisma/trustpilot-scraper/internal/config"
	"github.com/awang-karisma/trustpilot-scraper/internal/database"
	"github.com/awang-karisma/trustpilot-scraper/internal/scraper"
	"github.com/awang-karisma/trustpilot-scraper/internal/webhook"
	"log"
	"strings"
	"time"
)

func main() {
	// 1. Load Config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Connect to Database
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 3. Initialize Scraper
	s := scraper.NewScraper(cfg.TrustpilotURL)

	// 4. Scrape
	result, err := s.Scrape()
	if err != nil {
		log.Fatalf("Failed to scrape: %v", err)
	}
	log.Printf("Scraped %d reviews", len(result.Reviews))

	// 5. Save Summary
	if err := db.Create(&result.Summary).Error; err != nil {
		log.Printf("Failed to save summary: %v", err)
	}

	// 6. Process Reviews
	for _, review := range result.Reviews {
		// Check if review already exists
		var existing database.Review
		err := db.Where("review_id = ?", review.ReviewID).First(&existing).Error

		if err == nil {
			log.Printf("Review %s already exists in database, skipping", review.ReviewID)
			continue
		}

		// Save new review
		if err := db.Create(&review).Error; err != nil {
			log.Printf("Failed to save review %s: %v", review.ReviewID, err)
			continue
		}
		log.Printf("Saved new review: %s (%d stars)", review.ReviewID, review.Rating)
	}

	// 7. Find the worst review of the day
	var worstReview database.Review
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	err = db.Where("date >= ?", startOfDay).Order("rating ASC, date DESC").First(&worstReview).Error
	if err != nil {
		log.Printf("No reviews found for today yet: %v", err)
		// We still send a success webhook but with empty review data if desired,
		// or we can just use a placeholder.
	}

	// 8. Send Success Webhook
	if cfg.WebhookURL != "" {
		// Extract website name from URL
		website := cfg.TrustpilotURL
		parts := strings.Split(cfg.TrustpilotURL, "/")
		if len(parts) > 0 {
			website = parts[len(parts)-1]
		}

		webhookSvc := webhook.NewWebhookService(cfg.WebhookURL, cfg.WebhookTemplate)
		log.Printf("Sending success run webhook with worst review of the day: %s", worstReview.ReviewID)
		if err := webhookSvc.Send(website, worstReview, result.Summary); err != nil {
			log.Printf("Failed to send success webhook: %v", err)
		} else {
			log.Printf("Success webhook sent successfully")
		}
	} else {
		log.Println("Webhook URL is not configured, skipping success notification")
	}

	log.Println("Scraping completed successfully")
}
