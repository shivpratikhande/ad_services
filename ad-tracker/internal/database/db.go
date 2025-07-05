package database

import (
	"fmt"
	"time"

	"ad-tracking-system/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func SetupDatabase(databaseURL string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	fmt.Println("Using DSN:", databaseURL)

	if err != nil {
		return nil, err
	}

	// Auto-migrate schemas
	if err := db.AutoMigrate(&models.Ad{}, &models.ClickEvent{}); err != nil {
		return nil, err
	}

	// Set connection pool settings
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}

func SeedDatabase(db *gorm.DB) error {
	// Check if we already have ads
	var count int64
	db.Model(&models.Ad{}).Count(&count)
	if count > 0 {
		return nil
	}

	// Create sample ads
	sampleAds := []models.Ad{
		{
			ImageURL:  "https://example.com/ad1.jpg",
			TargetURL: "https://example.com/product1",
			Title:     "Amazing Product 1",
			Active:    true,
		},
		{
			ImageURL:  "https://example.com/ad2.jpg",
			TargetURL: "https://example.com/product2",
			Title:     "Great Service 2",
			Active:    true,
		},
		{
			ImageURL:  "https://example.com/ad3.jpg",
			TargetURL: "https://example.com/product3",
			Title:     "Special Offer 3",
			Active:    true,
		},
	}

	return db.Create(&sampleAds).Error
}
