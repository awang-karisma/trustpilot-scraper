package queue

import (
	"context"
	"time"
)

// Job represents a scrape job in the queue
type Job struct {
	ID          string    `json:"id"`
	WebsiteID   uint      `json:"website_id"`
	Priority    int       `json:"priority"` // Higher = more urgent
	Attempts    int       `json:"attempts"`
	MaxAttempts int       `json:"max_attempts"`
	CreatedAt   time.Time `json:"created_at"`
	ScheduledAt time.Time `json:"scheduled_at"` // When job should be processed
}

// JobStatus represents the status of a job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
)

// Priority levels
const (
	PriorityLow      = 1
	PriorityNormal   = 5
	PriorityHigh     = 10
	PriorityCritical = 20
)

// Queue defines the interface for a job queue
type Queue interface {
	// Enqueue adds a job to the queue
	Enqueue(job Job) error

	// Dequeue retrieves the next available job
	// Returns nil if no jobs available
	Dequeue(ctx context.Context) (*Job, error)

	// Ack acknowledges successful job completion
	Ack(jobID string) error

	// Nack marks job as failed, optionally requeueing for retry
	Nack(jobID string, retry bool) error

	// Size returns the number of pending jobs
	Size() int

	// Close shuts down the queue
	Close() error
}

// InFlightJob tracks a job being processed
type InFlightJob struct {
	Job       *Job
	StartedAt time.Time
}
