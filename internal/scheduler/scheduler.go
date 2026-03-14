package scheduler

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"

	"github.com/awang-karisma/trustpilot-scraper/internal/config"
	"github.com/awang-karisma/trustpilot-scraper/internal/database"
	"github.com/awang-karisma/trustpilot-scraper/internal/queue"
)

// Scheduler manages cron-based job scheduling for websites
type Scheduler struct {
	db                  *gorm.DB
	queue               queue.Queue
	cron                *cron.Cron
	config              *config.ServiceConfig
	logger              *slog.Logger
	entries             map[uint]cron.EntryID   // website ID -> cron entry ID
	notificationEntries map[string]cron.EntryID // channel ID -> cron entry ID
	mu                  sync.RWMutex
}

// NewScheduler creates a new scheduler
func NewScheduler(db *gorm.DB, q queue.Queue, cfg *config.ServiceConfig, logger *slog.Logger) *Scheduler {
	return &Scheduler{
		db:                  db,
		queue:               q,
		config:              cfg,
		logger:              logger,
		entries:             make(map[uint]cron.EntryID),
		notificationEntries: make(map[string]cron.EntryID),
		cron: cron.New(cron.WithLocation(time.UTC), cron.WithParser(cron.NewParser(
			cron.SecondOptional|cron.Minute|cron.Hour|cron.Dom|cron.Month|cron.Dow,
		))),
	}
}

// Start begins the scheduler
func (s *Scheduler) Start() error {
	s.logger.Info("Starting scheduler")

	// Check if website table is empty
	var count int64
	if err := s.db.Model(&database.Website{}).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to count websites: %w", err)
	}

	if count == 0 {
		s.logger.Info("No websites in database, scheduler will not run any jobs")
		// Start cron anyway to keep scheduler running
		s.cron.Start()
		s.logger.Info("Scheduler started with no websites")
		return nil
	}

	// Load all enabled websites and schedule them
	var websites []database.Website
	if err := s.db.Where("enabled = ?", true).Find(&websites).Error; err != nil {
		return err
	}

	if len(websites) == 0 {
		s.logger.Info("No enabled websites found, scheduler will not run any jobs")
		s.cron.Start()
		s.logger.Info("Scheduler started with no enabled websites")
		return nil
	}

	for _, website := range websites {
		if err := s.ScheduleWebsite(website); err != nil {
			s.logger.Error("Failed to schedule website", "website_id", website.ID, "error", err)
		}
	}

	// Load all enabled notification channels and schedule them
	var channels []database.NotificationChannel
	if err := s.db.Where("enabled = ?", true).Find(&channels).Error; err != nil {
		s.logger.Error("Failed to load notification channels", "error", err)
	} else {
		for _, channel := range channels {
			if err := s.ScheduleNotificationChannel(channel); err != nil {
				s.logger.Error("Failed to schedule notification channel", "channel_id", channel.ID, "error", err)
			}
		}
	}

	// Start cron
	s.cron.Start()

	s.logger.Info("Scheduler started", "websites_scheduled", len(websites), "channels_scheduled", len(channels))
	return nil
}

// Stop gracefully stops the scheduler
func (s *Scheduler) Stop() error {
	s.logger.Info("Stopping scheduler")

	// Stop cron (waits for running jobs to complete)
	ctx := s.cron.Stop()
	<-ctx.Done()

	s.logger.Info("Scheduler stopped")
	return nil
}

// ScheduleWebsite adds or updates a website's schedule
func (s *Scheduler) ScheduleWebsite(website database.Website) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove existing schedule if any
	if entryID, exists := s.entries[website.ID]; exists {
		s.cron.Remove(entryID)
		delete(s.entries, website.ID)
	}

	// Skip if not enabled
	if !website.Enabled {
		s.logger.Debug("Website disabled, skipping schedule", "website_id", website.ID, "name", website.Name)
		return nil
	}

	// Add new schedule
	entryID, err := s.cron.AddFunc(website.Schedule, func() {
		s.enqueueJob(website.ID, queue.PriorityNormal)
	})
	if err != nil {
		s.logger.Error("Failed to schedule website", "website_id", website.ID, "schedule", website.Schedule, "error", err)
		return err
	}

	s.entries[website.ID] = entryID
	s.logger.Info("Website scheduled", "website_id", website.ID, "name", website.Name, "schedule", website.Schedule)

	return nil
}

// UnscheduleWebsite removes a website's schedule
func (s *Scheduler) UnscheduleWebsite(websiteID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, exists := s.entries[websiteID]; exists {
		s.cron.Remove(entryID)
		delete(s.entries, websiteID)
		s.logger.Info("Website unscheduled", "website_id", websiteID)
	}
}

// TriggerImmediate enqueues a high-priority job for immediate processing
func (s *Scheduler) TriggerImmediate(websiteID uint) error {
	return s.enqueueJob(websiteID, queue.PriorityHigh)
}

// enqueueJob creates and enqueues a scrape job
func (s *Scheduler) enqueueJob(websiteID uint, priority int) error {
	// Get website
	var website database.Website
	if err := s.db.First(&website, websiteID).Error; err != nil {
		s.logger.Error("Failed to get website for job", "website_id", websiteID, "error", err)
		return err
	}

	// Create job
	job := queue.Job{
		WebsiteID:   websiteID,
		Priority:    priority,
		MaxAttempts: s.config.MaxRetries,
		CreatedAt:   time.Now(),
		ScheduledAt: time.Now(),
	}

	// Enqueue
	if err := s.queue.Enqueue(job); err != nil {
		s.logger.Error("Failed to enqueue job", "website_id", websiteID, "error", err)
		return err
	}

	s.logger.Info("Job enqueued", "website_id", websiteID, "name", website.Name, "priority", priority)
	return nil
}

// GetScheduledWebsites returns the list of scheduled website IDs
func (s *Scheduler) GetScheduledWebsites() []uint {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]uint, 0, len(s.entries))
	for id := range s.entries {
		ids = append(ids, id)
	}
	return ids
}

// Reload reloads all website schedules from database
func (s *Scheduler) Reload() error {
	s.logger.Info("Reloading website schedules")

	// Clear all existing schedules
	s.mu.Lock()
	for _, entryID := range s.entries {
		s.cron.Remove(entryID)
	}
	for _, entryID := range s.notificationEntries {
		s.cron.Remove(entryID)
	}
	s.entries = make(map[uint]cron.EntryID)
	s.notificationEntries = make(map[string]cron.EntryID)
	s.mu.Unlock()

	// Reload from database
	return s.Start()
}

// LoadNotificationChannels loads all enabled notification channels from database
func (s *Scheduler) LoadNotificationChannels() error {
	var channels []database.NotificationChannel
	if err := s.db.Where("enabled = ?", true).Find(&channels).Error; err != nil {
		s.logger.Error("Failed to load notification channels", "error", err)
		return err
	}

	s.logger.Info("Loading notification channels", "count", len(channels))

	for _, channel := range channels {
		if err := s.ScheduleNotificationChannel(channel); err != nil {
			s.logger.Error("Failed to schedule notification channel", "channel_id", channel.ID, "error", err)
		}
	}

	return nil
}

// ScheduleNotificationChannel schedules a notification channel
func (s *Scheduler) ScheduleNotificationChannel(channel database.NotificationChannel) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove existing schedule if any
	if entryID, exists := s.notificationEntries[channel.ID]; exists {
		s.cron.Remove(entryID)
		delete(s.notificationEntries, channel.ID)
	}

	// Skip if not enabled
	if !channel.Enabled {
		s.logger.Debug("Notification channel disabled, skipping schedule", "channel_id", channel.ID, "name", channel.Name)
		return nil
	}

	// Add new schedule
	entryID, err := s.cron.AddFunc(channel.Schedule, func() {
		s.enqueueNotificationJob(channel.ID, queue.PriorityNormal)
	})
	if err != nil {
		s.logger.Error("Failed to schedule notification channel", "channel_id", channel.ID, "schedule", channel.Schedule, "error", err)
		return err
	}

	s.notificationEntries[channel.ID] = entryID
	s.logger.Info("Notification channel scheduled", "channel_id", channel.ID, "name", channel.Name, "schedule", channel.Schedule)

	return nil
}

// UnscheduleNotificationChannel removes a notification channel from scheduler
func (s *Scheduler) UnscheduleNotificationChannel(channelID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, exists := s.notificationEntries[channelID]; exists {
		s.cron.Remove(entryID)
		delete(s.notificationEntries, channelID)
		s.logger.Info("Notification channel unscheduled", "channel_id", channelID)
	}
}

// TriggerNotificationImmediate triggers an immediate notification job
func (s *Scheduler) TriggerNotificationImmediate(channelID string) error {
	return s.enqueueNotificationJob(channelID, queue.PriorityHigh)
}

// enqueueNotificationJob creates and enqueues a notification job
func (s *Scheduler) enqueueNotificationJob(channelID string, priority int) error {
	// Get channel
	var channel database.NotificationChannel
	if err := s.db.First(&channel, "id = ?", channelID).Error; err != nil {
		s.logger.Error("Failed to get notification channel for job", "channel_id", channelID, "error", err)
		return err
	}

	// Create job
	job := queue.Job{
		Type:        "notification",
		ChannelID:   channelID,
		Priority:    priority,
		MaxAttempts: s.config.MaxRetries,
		CreatedAt:   time.Now(),
		ScheduledAt: time.Now(),
	}

	// Enqueue
	if err := s.queue.Enqueue(job); err != nil {
		s.logger.Error("Failed to enqueue notification job", "channel_id", channelID, "error", err)
		return err
	}

	s.logger.Info("Notification job enqueued", "channel_id", channelID, "name", channel.Name, "priority", priority)
	return nil
}
