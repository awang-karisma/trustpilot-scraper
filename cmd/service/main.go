package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/awang-karisma/trustpilot-scraper/internal/api"
	"github.com/awang-karisma/trustpilot-scraper/internal/config"
	"github.com/awang-karisma/trustpilot-scraper/internal/database"
	"github.com/awang-karisma/trustpilot-scraper/internal/queue"
	"github.com/awang-karisma/trustpilot-scraper/internal/scheduler"
	"github.com/awang-karisma/trustpilot-scraper/internal/website"
	"github.com/awang-karisma/trustpilot-scraper/internal/worker"

	_ "github.com/awang-karisma/trustpilot-scraper/docs" // Import docs for Swaggo
)

// @title Trustpilot Scraper API
// @version 1.0
// @description This is a Trustpilot Scraper API server.
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.email support@example.com
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @BasePath /

func main() {
	// Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("Starting Trustpilot Scraper Service")

	// 1. Load configuration
	cfg, err := config.LoadServiceConfig()
	if err != nil {
		logger.Error("Failed to load config", "error", err)
		os.Exit(1)
	}
	logger.Info("Configuration loaded",
		"worker_count", cfg.WorkerCount,
		"api_port", cfg.APIPort,
		"default_schedule", cfg.DefaultSchedule,
	)

	// 2. Connect to database
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	logger.Info("Database connected")

	// 3. Initialize database (auto-migration)
	initManager := database.NewInitManager(db, logger)

	// Check if we should drop tables first (safety feature)
	if cfg.DropTablesOnStart {
		logger.Warn("DROP_TABLES_ON_START is enabled, dropping all existing tables")
		if err := initManager.DropAllTables(); err != nil {
			logger.Error("Failed to drop tables", "error", err)
			os.Exit(1)
		}
	}

	// Run auto-migration
	if err := initManager.AutoMigrate(); err != nil {
		logger.Error("Failed to run auto-migration", "error", err)
		os.Exit(1)
	}
	logger.Info("Database migration completed")

	// 4. Initialize websites from environment variables
	// This will only add websites if they don't exist in the database
	websiteManager := website.NewManager(db, logger)
	if err := websiteManager.InitializeFromEnv(cfg.TrustpilotURL, cfg.DefaultSchedule); err != nil {
		logger.Error("Failed to initialize websites from environment", "error", err)
		// Continue anyway - websites can be added via API
		logger.Warn("Continuing without website initialization")
	}

	// 5. Initialize queue
	q := queue.NewMemoryQueue()
	logger.Info("Queue initialized", "queue_size", cfg.QueueSize)

	// 6. Create scheduler
	sched := scheduler.NewScheduler(db, q, cfg, logger)

	// 7. Create worker pool
	pool := worker.NewPool(db, q, cfg, logger)

	// 7. Create API server
	server := api.NewServer(cfg, db, sched, logger)

	// Setup graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// 8. Start components
	// Start worker pool first (consumers)
	pool.Start()

	// Start scheduler (producer)
	if err := sched.Start(); err != nil {
		logger.Error("Failed to start scheduler", "error", err)
		cancel()
	}

	// Start API server if enabled
	var apiErrors chan error
	if cfg.APIEnabled {
		apiErrors = make(chan error, 1)
		go func() {
			if err := server.Start(); err != nil && err != http.ErrServerClosed {
				apiErrors <- err
			}
		}()
	} else {
		logger.Info("API server disabled by configuration")
		apiErrors = nil
	}

	logger.Info("Service started",
		"workers", cfg.WorkerCount,
		"api_port", cfg.APIPort,
		"api_enabled", cfg.APIEnabled,
	)

	// 9. Wait for shutdown signal or API error
	if cfg.APIEnabled {
		select {
		case <-ctx.Done():
			logger.Info("Shutdown signal received")
		case err := <-apiErrors:
			logger.Error("API server error", "error", err)
			cancel()
		}
	} else {
		// Only wait for shutdown signal if API is disabled
		<-ctx.Done()
		logger.Info("Shutdown signal received")
	}

	// Start API server (if enabled)
	if cfg.APIEnabled {
		apiErrors := make(chan error, 1)
		go func() {
			if err := server.Start(); err != nil && err != http.ErrServerClosed {
				apiErrors <- err
			}
		}()
	} else {
		logger.Info("API server disabled by configuration")
	}

	logger.Info("Service started",
		"workers", cfg.WorkerCount,
		"api_port", cfg.APIPort,
		"api_enabled", cfg.APIEnabled,
	)

	// 9. Wait for shutdown signal or error
	select {
	case <-ctx.Done():
		logger.Info("Shutdown signal received")
	case err := <-apiErrors:
		logger.Error("API server error", "error", err)
	}

	// 10. Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(),
		time.Duration(cfg.ShutdownTimeoutSec)*time.Second)
	defer shutdownCancel()

	// Stop API server
	if err := server.Shutdown(); err != nil {
		logger.Error("Failed to shutdown API server", "error", err)
	}

	// Stop scheduler
	if err := sched.Stop(); err != nil {
		logger.Error("Failed to stop scheduler", "error", err)
	}

	// Stop worker pool
	if err := pool.Stop(); err != nil {
		logger.Error("Failed to stop worker pool", "error", err)
	}

	// Close queue
	if err := q.Close(); err != nil {
		logger.Error("Failed to close queue", "error", err)
	}

	// Wait for shutdown context
	<-shutdownCtx.Done()
	if shutdownCtx.Err() == context.DeadlineExceeded {
		logger.Warn("Shutdown timeout exceeded")
	}

	logger.Info("Service stopped")
}
