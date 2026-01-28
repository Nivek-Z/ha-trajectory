package repository

import (
	"fmt"
	"time"

	"ha-trajectory/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ConnectWithRetry opens a Postgres connection with retry.
func ConnectWithRetry(dsn string, attempts int, delay time.Duration) (*gorm.DB, error) {
	var lastErr error
	for i := 1; i <= attempts; i++ {
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			if err := bootstrap(db); err != nil {
				return nil, err
			}
			return db, nil
		}

		lastErr = err
		time.Sleep(delay)
	}

	return nil, fmt.Errorf("db connect failed after %d attempts: %w", attempts, lastErr)
}

func bootstrap(db *gorm.DB) error {
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS postgis").Error; err != nil {
		return err
	}

	return db.AutoMigrate(&models.TrackPoint{})
}