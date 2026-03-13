package worker

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/awang-karisma/trustpilot-scraper/internal/config"
	"github.com/awang-karisma/trustpilot-scraper/internal/queue"
	"github.com/awang-karisma/trustpilot-scraper/internal/webhook"
)

// Pool manages a pool of workers
type Pool struct {
	workers    []*Worker
	queue      queue.Queue
	db         *gorm.DB
	config     *config.ServiceConfig
	webhook    *webhook.WebhookService
	logger     *slog.Logger
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	activeJobs sync.Map
}

// NewPool creates a new worker pool
func NewPool(db *gorm.DB, q queue.Queue, cfg *config.ServiceConfig, wh *webhook.WebhookService, logger *slog.Logger) *Pool {
	return &Pool{
		db:      db,
		queue:   q,
		config:  cfg,
		webhook: wh,
		logger:  logger,
	}
}

// Start initializes and starts all workers
func (p *Pool) Start() {
	p.ctx, p.cancel = context.WithCancel(context.Background())

	// Create workers
	p.workers = make([]*Worker, p.config.WorkerCount)
	for i := 0; i < p.config.WorkerCount; i++ {
		p.workers[i] = NewWorker(i+1, p.db, p.queue, p.config, p.webhook, p.logger)
	}

	// Start workers
	for _, worker := range p.workers {
		p.wg.Add(1)
		go func(w *Worker) {
			defer p.wg.Done()
			w.Start(p.ctx)
		}(worker)
	}

	p.logger.Info("Worker pool started", "worker_count", p.config.WorkerCount)
}

// Stop gracefully stops all workers
func (p *Pool) Stop() error {
	p.logger.Info("Stopping worker pool")

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
		p.logger.Info("Worker pool stopped gracefully")
		return nil
	case <-time.After(timeout):
		p.logger.Warn("Worker pool stop timeout, some workers may not have finished")
		return nil
	}
}

// ActiveCount returns the number of active workers
func (p *Pool) ActiveCount() int {
	return p.config.WorkerCount
}

// QueueSize returns the current queue size
func (p *Pool) QueueSize() int {
	return p.queue.Size()
}

// Stats returns pool statistics
func (p *Pool) Stats() PoolStats {
	return PoolStats{
		Workers:   p.config.WorkerCount,
		QueueSize: p.queue.Size(),
	}
}

// PoolStats represents worker pool statistics
type PoolStats struct {
	Workers   int `json:"workers"`
	QueueSize int `json:"queue_size"`
}
