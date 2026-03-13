package database

import (
	"fmt"
	"log/slog"

	"gorm.io/gorm"
)

// InitManager handles database initialization and migrations
type InitManager struct {
	db     *gorm.DB
	logger *slog.Logger
}

// NewInitManager creates a new initialization manager
func NewInitManager(db *gorm.DB, logger *slog.Logger) *InitManager {
	return &InitManager{
		db:     db,
		logger: logger,
	}
}

// AutoMigrate performs automatic migration without SQL files
func (m *InitManager) AutoMigrate() error {
	m.logger.Info("Starting automatic migration")

	// Auto-migrate all models
	tables := []interface{}{
		&Website{},
		&Review{},
		&WebsiteRating{},
		&ScrapeJob{},
		&Summary{},
	}

	for _, table := range tables {
		if err := m.db.AutoMigrate(table); err != nil {
			return fmt.Errorf("failed to migrate %T: %w", table, err)
		}
	}

	m.logger.Info("Automatic migration completed successfully")
	return nil
}
