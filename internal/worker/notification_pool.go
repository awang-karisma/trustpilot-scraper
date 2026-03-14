package worker

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/awang-karisma/trustpilot-scraper/internal/config"
	"github.com/awang-karisma/trustpilot-scraper/internal/queue"
)

// NotificationPool manages a pool of notification workers
type NotificationPool struct {
	workers []*NotificationWorker
	queue   queue.Queue
	db      *gorm.DB
	config  *config.ServiceConfig
	logger  *slog.Logger
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// NewNotificationPool creates a new notification worker pool
func NewNotificationPool(db *gorm.DB, q queue.Queue, cfg *config.ServiceConfig, logger *slog.Logger) *NotificationPool {
	return &NotificationPool{
		db:     db,
		queue:  q,
		config: cfg,
		logger: logger,
	}
}

// Start initializes and starts all notification workers
func (p *NotificationPool) Start() {
	p.ctx, p.cancel = context.WithCancel(context.Background())

	// Create notification workers (use same count as scrape workers)
	p.workers = make([]*NotificationWorker, p.config.WorkerCount)
	for i := 0; i < p.config.WorkerCount; i++ {
		p.workers[i] = NewNotificationWorker(i+1, p.db, p.queue, p.config, p.logger)
	}

	// Start workers
	for _, worker := range p.workers {
		p.wg.Add(1)
		go func(w *NotificationWorker) {
			defer p.wg.Done()
			w.Start(p.ctx)
		}(worker)
	}

	p.logger.Info("Notification worker pool started", "worker_count", p.config.WorkerCount)
}

// Stop gracefully stops all notification workers
func (p *NotificationPool) Stop() error {
	p.logger.Info("Stopping notification worker pool")

	// Cancel context to signal workers to stop
	p.cancel()

	// Wait for workers with timeout
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	timeout := time.Duration(p.config.ShutdownTimeoutSec) * time.Second
	select {
	case <-done:
		p.logger.Info("Notification worker pool stopped gracefully")
		return nil
	case <-time.After(timeout):
		p.logger.Warn("Notification worker pool stop timeout, some workers may not have finished")
		return nil
	}
}

// ActiveCount returns the number of active notification workers
func (p *NotificationPool) ActiveCount() int {
	return p.config.WorkerCount
}

// Stats returns pool statistics
func (p *NotificationPool) Stats() NotificationPoolStats {
	return NotificationPoolStats{
		Workers:   p.config.WorkerCount,
		QueueSize: p.queue.Size(),
	}
}

// NotificationPoolStats represents notification worker pool statistics
type NotificationPoolStats struct {
	Workers   int `json:"workers"`
	QueueSize int `json:"queue_size"`
}
