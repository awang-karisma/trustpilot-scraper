package queue

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MemoryQueue is an in-memory implementation of the Queue interface
type MemoryQueue struct {
	mu       sync.RWMutex
	pending  []*Job          // Jobs waiting to be processed
	inFlight map[string]*Job // Jobs currently being processed
	delayed  []*Job          // Jobs scheduled for future processing
	stopped  bool
	stopChan chan struct{}
}

// NewMemoryQueue creates a new in-memory queue
func NewMemoryQueue() *MemoryQueue {
	return &MemoryQueue{
		pending:  make([]*Job, 0),
		inFlight: make(map[string]*Job),
		delayed:  make([]*Job, 0),
		stopChan: make(chan struct{}),
	}
}

// Enqueue adds a job to the queue
func (q *MemoryQueue) Enqueue(job Job) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.stopped {
		return ErrQueueStopped
	}

	// Generate ID if not set
	if job.ID == "" {
		job.ID = uuid.New().String()
	}

	// Set defaults
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}
	if job.ScheduledAt.IsZero() {
		job.ScheduledAt = time.Now()
	}
	if job.MaxAttempts == 0 {
		job.MaxAttempts = 3
	}

	// If scheduled for future, add to delayed queue
	if job.ScheduledAt.After(time.Now()) {
		q.delayed = append(q.delayed, &job)
		return nil
	}

	// Add to pending queue based on priority
	q.insertByPriority(&job)
	return nil
}

// Dequeue retrieves the next available job
func (q *MemoryQueue) Dequeue(ctx context.Context) (*Job, error) {
	for {
		q.mu.Lock()

		if q.stopped {
			q.mu.Unlock()
			return nil, ErrQueueStopped
		}

		// Move delayed jobs that are ready to pending
		q.moveDelayedJobs()

		if len(q.pending) > 0 {
			job := q.pending[0]
			q.pending = q.pending[1:]
			q.inFlight[job.ID] = job
			q.mu.Unlock()
			return job, nil
		}

		q.mu.Unlock()

		// Wait for new jobs or context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(100 * time.Millisecond):
			// Poll again
			continue
		}
	}
}

// Ack acknowledges successful job completion
func (q *MemoryQueue) Ack(jobID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, exists := q.inFlight[jobID]; !exists {
		return ErrJobNotFound
	}

	delete(q.inFlight, jobID)
	return nil
}

// Nack marks job as failed, optionally requeueing for retry
func (q *MemoryQueue) Nack(jobID string, retry bool) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	job, exists := q.inFlight[jobID]
	if !exists {
		return ErrJobNotFound
	}

	delete(q.inFlight, jobID)

	if retry && job.Attempts < job.MaxAttempts {
		job.Attempts++
		// Exponential backoff: 5s, 25s, 125s
		backoff := time.Duration(5) * time.Second
		for i := 1; i < job.Attempts; i++ {
			backoff *= 5
		}
		job.ScheduledAt = time.Now().Add(backoff)
		q.delayed = append(q.delayed, job)
	}

	return nil
}

// Size returns the number of pending jobs
func (q *MemoryQueue) Size() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.pending) + len(q.delayed)
}

// Close shuts down the queue
func (q *MemoryQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.stopped = true
	close(q.stopChan)
	return nil
}

// insertByPriority inserts a job into the pending queue by priority (highest first)
func (q *MemoryQueue) insertByPriority(job *Job) {
	// Find insertion point
	i := 0
	for ; i < len(q.pending); i++ {
		if q.pending[i].Priority < job.Priority {
			break
		}
	}

	// Insert at position
	if i == len(q.pending) {
		q.pending = append(q.pending, job)
	} else {
		q.pending = append(q.pending[:i], append([]*Job{job}, q.pending[i:]...)...)
	}
}

// moveDelayedJobs moves ready delayed jobs to pending queue
func (q *MemoryQueue) moveDelayedJobs() {
	now := time.Now()
	ready := make([]*Job, 0)
	remaining := make([]*Job, 0)

	for _, job := range q.delayed {
		if job.ScheduledAt.Before(now) || job.ScheduledAt.Equal(now) {
			ready = append(ready, job)
		} else {
			remaining = append(remaining, job)
		}
	}

	q.delayed = remaining

	for _, job := range ready {
		q.insertByPriority(job)
	}
}

// Queue errors
var (
	ErrQueueStopped = &QueueError{Message: "queue is stopped"}
	ErrJobNotFound  = &QueueError{Message: "job not found"}
)

// QueueError represents a queue error
type QueueError struct {
	Message string
}

func (e *QueueError) Error() string {
	return e.Message
}
