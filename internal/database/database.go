package database

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto-migrate
	err = db.AutoMigrate(&Website{}, &Review{}, &WebsiteRating{}, &ScrapeJob{}, &Summary{})
	if err != nil {
		return nil, fmt.Errorf("failed to auto-migrate database: %w", err)
	}

	return db, nil
}
