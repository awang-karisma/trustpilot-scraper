package main

import (
	"github.com/awang-karisma/trustpilot-scraper/internal/config"
	"github.com/awang-karisma/trustpilot-scraper/internal/database"
	"github.com/awang-karisma/trustpilot-scraper/internal/scraper"
	"log"
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
	}

	log.Println("Scraping completed successfully")
}
