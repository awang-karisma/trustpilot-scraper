-- Migration: 001_initial_schema
-- Description: Create initial schema for trustpilot scraper service
-- Created: 2026-03-13

-- Enable UUID extension if not exists
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Websites table: Target websites to scrape
CREATE TABLE IF NOT EXISTS websites (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    base_url VARCHAR(2048) NOT NULL,
    schedule VARCHAR(100) NOT NULL DEFAULT '*/30 * * * *',
    enabled BOOLEAN DEFAULT TRUE,
    last_scraped TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Reviews table: Scraped reviews from Trustpilot
CREATE TABLE IF NOT EXISTS reviews (
    id BIGSERIAL PRIMARY KEY,
    review_id VARCHAR(255) NOT NULL UNIQUE,
    website_id BIGINT NOT NULL REFERENCES websites(id) ON DELETE CASCADE,
    reviewer VARCHAR(255),
    title TEXT,
    content TEXT,
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    date TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Website ratings table: Rating snapshots over time
CREATE TABLE IF NOT EXISTS website_ratings (
    id BIGSERIAL PRIMARY KEY,
    website_id BIGINT NOT NULL REFERENCES websites(id) ON DELETE CASCADE,
    rating DECIMAL(3, 2) NOT NULL,
    count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Scrape jobs table: Track scraping job executions
CREATE TABLE IF NOT EXISTS scrape_jobs (
    id BIGSERIAL PRIMARY KEY,
    website_id BIGINT NOT NULL REFERENCES websites(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    error TEXT,
    reviews_found INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for common queries

-- Websites
CREATE INDEX idx_websites_enabled ON websites(enabled);
CREATE INDEX idx_websites_last_scraped ON websites(last_scraped);

-- Reviews
CREATE INDEX idx_reviews_website_id ON reviews(website_id);
CREATE INDEX idx_reviews_rating ON reviews(rating);
CREATE INDEX idx_reviews_date ON reviews(date);
CREATE INDEX idx_reviews_deleted_at ON reviews(deleted_at);
CREATE INDEX idx_reviews_website_rating ON reviews(website_id, rating);
CREATE INDEX idx_reviews_website_date ON reviews(website_id, date DESC);

-- Website ratings
CREATE INDEX idx_website_ratings_website_id ON website_ratings(website_id);
CREATE INDEX idx_website_ratings_created_at ON website_ratings(created_at DESC);
CREATE INDEX idx_website_ratings_website_created ON website_ratings(website_id, created_at DESC);

-- Scrape jobs
CREATE INDEX idx_scrape_jobs_website_id ON scrape_jobs(website_id);
CREATE INDEX idx_scrape_jobs_status ON scrape_jobs(status);
CREATE INDEX idx_scrape_jobs_created_at ON scrape_jobs(created_at DESC);
CREATE INDEX idx_scrape_jobs_website_status ON scrape_jobs(website_id, status);

-- Summary table (for backward compatibility)
CREATE TABLE IF NOT EXISTS summaries (
    id BIGSERIAL PRIMARY KEY,
    rating DECIMAL(3, 2) NOT NULL,
    count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply trigger to websites table
DROP TRIGGER IF EXISTS update_websites_updated_at ON websites;
CREATE TRIGGER update_websites_updated_at
    BEFORE UPDATE ON websites
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Apply trigger to reviews table
DROP TRIGGER IF EXISTS update_reviews_updated_at ON reviews;
CREATE TRIGGER update_reviews_updated_at
    BEFORE UPDATE ON reviews
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
