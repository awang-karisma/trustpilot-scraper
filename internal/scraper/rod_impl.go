package scraper

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/awang-karisma/trustpilot-scraper/internal/config"
	"github.com/awang-karisma/trustpilot-scraper/internal/database"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// Scraper provides Go Rod scraping capabilities
type Scraper struct {
	browser *rod.Browser
	config  *config.RodConfig
	url     string
}

// NewScraper creates a new Scraper instance
func NewScraper(url string) *Scraper {
	cfg := config.LoadRodConfig()
	return &Scraper{
		url:    url,
		config: cfg,
	}
}

// Scrape performs scraping using Go Rod
func (r *Scraper) Scrape() (*ScrapeResult, error) {
	log.Printf("Starting Rod scraping for URL: %s", r.url)

	result := &ScrapeResult{
		Reviews: []database.Review{},
	}

	// Try to connect to any available Chrome
	browser, err := r.connectToChrome()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Chrome: %w", err)
	}

	r.browser = browser
	defer r.browser.Close()

	log.Println("Successfully connected to Chrome")

	// Create and navigate to page
	page := browser.MustPage(r.url)
	err = page.WaitLoad()
	if err != nil {
		return nil, fmt.Errorf("page load failed: %w", err)
	}

	log.Println("Page loaded successfully")

	// Extract Summary (Rating and Count)
	summaryEl, err := page.Element("div[class*='styles_rating'], div[data-testid='rating']")
	if err == nil {
		// Extract rating
		ratingEl, err := summaryEl.Element("p[data-rating-typography='true'], span[data-testid='rating-score']")
		if err == nil {
			ratingText, _ := ratingEl.Text()
			rating, err := strconv.ParseFloat(ratingText, 64)
			if err == nil {
				result.Summary.Rating = rating
				log.Printf("Extracted rating: %.1f", rating)
			}
		}

		// Extract review count
		fullText, _ := summaryEl.Text()
		if strings.Contains(fullText, " reviews") {
			idx := strings.Index(fullText, " reviews")
			if idx > 0 {
				reviewCountStr := cleanReviewCount(fullText[:idx])
				count, err := parseReviewCount(reviewCountStr)
				if err == nil {
					result.Summary.Count = count
					log.Printf("Extracted review count: %d", count)
				}
			}
		}
	} else {
		log.Printf("Failed to find summary element: %v", err)
	}

	// Extract Reviews
	reviewCards, err := page.Elements("article[data-service-review-card-paper='true']")
	if err == nil {
		log.Printf("Found %d review cards", len(reviewCards))
		for _, card := range reviewCards {
			review := database.Review{}

			// Review ID - extract from review link href
			reviewLinkEl, err := card.Element("a[href*='/reviews/']")
			if err == nil {
				href, _ := reviewLinkEl.Attribute("href")
				if href != nil {
					// Extract ID from href like "/reviews/6879f42d7634eaeb833df2ff"
					parts := strings.Split(*href, "/reviews/")
					if len(parts) > 1 {
						review.ReviewID = parts[1]
					}
				}
			}

			// Reviewer
			reviewerEl, err := card.Element("span[data-consumer-name-typography='true']")
			if err == nil {
				review.Reviewer, _ = reviewerEl.Text()
			}

			// Title
			titleEl, err := card.Element("h2[data-service-review-title-typography='true']")
			if err == nil {
				review.Title, _ = titleEl.Text()
			}

			// Content
			contentEl, err := card.Element("p[data-service-review-text-typography='true']")
			if err == nil {
				review.Content, _ = contentEl.Text()
			}

			// Rating
			ratingImg, err := card.Element("img[class*='StarRating_starRating']")
			if err == nil {
				alt, _ := ratingImg.Attribute("alt")
				if alt != nil && strings.Contains(*alt, "out of 5") {
					// Alt text format: "Rated 1 out of 5 stars"
					// We need to extract the number after "Rated"
					parts := strings.Split(*alt, " ")
					if len(parts) >= 2 {
						r, err := strconv.Atoi(parts[1]) // Second part is the rating number
						if err == nil {
							review.Rating = r
						}
					}
				}
			}

			// Date
			dateEl, err := card.Element("time[data-service-review-date-time-ago='true']")
			if err == nil {
				dateStr, _ := dateEl.Attribute("datetime")
				if dateStr != nil {
					t, err := time.Parse(time.RFC3339, *dateStr)
					if err == nil {
						review.Date = t
					}
				}
			}

			result.Reviews = append(result.Reviews, review)
		}
	}

	result.Summary.CreatedAt = time.Now()
	log.Printf("Rod scraping completed: %.1f rating, %d reviews", result.Summary.Rating, result.Summary.Count)

	return result, nil
}

// ScrapeWithContext performs scraping with context support for cancellation and timeout
func (r *Scraper) ScrapeWithContext(ctx context.Context, url string) (*ScrapeResult, error) {
	// Use provided URL or fallback to scraper's URL
	if url == "" {
		url = r.url
	}

	log.Printf("Starting Rod scraping for URL: %s", url)

	result := &ScrapeResult{
		Reviews: []database.Review{},
	}

	// Try to connect to any available Chrome
	browser, err := r.connectToChrome()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Chrome: %w", err)
	}

	r.browser = browser
	defer r.browser.Close()

	log.Println("Successfully connected to Chrome")

	// Check context before proceeding
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Create and navigate to page with context
	page, err := browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}

	// Navigate with timeout
	page = page.Timeout(time.Duration(r.config.PageLoadTimeout) * time.Second)
	if err := page.Navigate(url); err != nil {
		return nil, fmt.Errorf("navigation failed: %w", err)
	}

	// Wait for page load with context
	waitChan := make(chan error, 1)
	go func() {
		waitChan <- page.WaitLoad()
	}()

	select {
	case <-ctx.Done():
		page.Close()
		return nil, ctx.Err()
	case err := <-waitChan:
		if err != nil {
			return nil, fmt.Errorf("page load failed: %w", err)
		}
	}

	log.Println("Page loaded successfully")

	// Check context before extracting data
	select {
	case <-ctx.Done():
		page.Close()
		return nil, ctx.Err()
	default:
	}

	// Extract Summary (Rating and Count)
	summaryEl, err := page.Element("div[class*='styles_rating'], div[data-testid='rating']")
	if err == nil {
		// Extract rating
		ratingEl, err := summaryEl.Element("p[data-rating-typography='true'], span[data-testid='rating-score']")
		if err == nil {
			ratingText, _ := ratingEl.Text()
			rating, err := strconv.ParseFloat(ratingText, 64)
			if err == nil {
				result.Summary.Rating = rating
				log.Printf("Extracted rating: %.1f", rating)
			}
		}

		// Extract review count
		fullText, _ := summaryEl.Text()
		if strings.Contains(fullText, " reviews") {
			idx := strings.Index(fullText, " reviews")
			if idx > 0 {
				reviewCountStr := cleanReviewCount(fullText[:idx])
				count, err := parseReviewCount(reviewCountStr)
				if err == nil {
					result.Summary.Count = count
					log.Printf("Extracted review count: %d", count)
				}
			}
		}
	} else {
		log.Printf("Failed to find summary element: %v", err)
	}

	// Check context before extracting reviews
	select {
	case <-ctx.Done():
		page.Close()
		return nil, ctx.Err()
	default:
	}

	// Extract Reviews
	reviewCards, err := page.Elements("article[data-service-review-card-paper='true']")
	if err == nil {
		log.Printf("Found %d review cards", len(reviewCards))
		for _, card := range reviewCards {
			// Check context during iteration
			select {
			case <-ctx.Done():
				page.Close()
				return nil, ctx.Err()
			default:
			}

			review := database.Review{}

			// Review ID - extract from review link href
			reviewLinkEl, err := card.Element("a[href*='/reviews/']")
			if err == nil {
				href, _ := reviewLinkEl.Attribute("href")
				if href != nil {
					// Extract ID from href like "/reviews/6879f42d7634eaeb833df2ff"
					parts := strings.Split(*href, "/reviews/")
					if len(parts) > 1 {
						review.ReviewID = parts[1]
					}
				}
			}

			// Reviewer
			reviewerEl, err := card.Element("span[data-consumer-name-typography='true']")
			if err == nil {
				review.Reviewer, _ = reviewerEl.Text()
			}

			// Title
			titleEl, err := card.Element("h2[data-service-review-title-typography='true']")
			if err == nil {
				review.Title, _ = titleEl.Text()
			}

			// Content
			contentEl, err := card.Element("p[data-service-review-text-typography='true']")
			if err == nil {
				review.Content, _ = contentEl.Text()
			}

			// Rating
			ratingImg, err := card.Element("img[class*='StarRating_starRating']")
			if err == nil {
				alt, _ := ratingImg.Attribute("alt")
				if alt != nil && strings.Contains(*alt, "out of 5") {
					// Alt text format: "Rated 1 out of 5 stars"
					// We need to extract the number after "Rated"
					parts := strings.Split(*alt, " ")
					if len(parts) >= 2 {
						r, err := strconv.Atoi(parts[1]) // Second part is the rating number
						if err == nil {
							review.Rating = r
						}
					}
				}
			}

			// Date
			dateEl, err := card.Element("time[data-service-review-date-time-ago='true']")
			if err == nil {
				dateStr, _ := dateEl.Attribute("datetime")
				if dateStr != nil {
					t, err := time.Parse(time.RFC3339, *dateStr)
					if err == nil {
						review.Date = t
					}
				}
			}

			result.Reviews = append(result.Reviews, review)
		}
	}

	result.Summary.CreatedAt = time.Now()
	log.Printf("Rod scraping completed: %.1f rating, %d reviews", result.Summary.Rating, result.Summary.Count)

	return result, nil
}

// connectToChrome attempts to connect to Chrome using rod launcher manager
func (r *Scraper) connectToChrome() (*rod.Browser, error) {
	if r.config.LauncherManagerURL == "" {
		return nil, fmt.Errorf("rod launcher manager URL not configured (CHROME_LAUNCHER_MANAGER_URL)")
	}

	log.Printf("Attempting to connect to Rod launcher manager: %s", r.config.LauncherManagerURL)
	l := launcher.MustNewManaged(r.config.LauncherManagerURL)

	browser := rod.New().Client(l.MustClient())
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to rod launcher manager at %s: %w", r.config.LauncherManagerURL, err)
	}

	log.Println("Successfully connected to Rod launcher manager")
	return browser, nil
}

// Helper functions

// cleanReviewCount extracts the numeric part from review count text
func cleanReviewCount(s string) string {
	re := regexp.MustCompile(`[\dK,.]+`)
	matches := re.FindAllString(s, -1)
	if len(matches) > 0 {
		return matches[len(matches)-1]
	}
	return s
}

// parseReviewCount converts review count string to integer
func parseReviewCount(s string) (int, error) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")

	if strings.HasSuffix(s, "K") || strings.HasSuffix(s, "k") {
		numStr := strings.TrimSuffix(strings.ToUpper(s), "K")
		num, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return 0, err
		}
		return int(num * 1000), nil
	}

	return strconv.Atoi(s)
}
