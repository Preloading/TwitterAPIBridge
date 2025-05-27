package db_controller

import (
	"log"
	"time"
)

var (
	inRamAnalytics  []AnalyticData
	analyticsTicker *time.Ticker
)

// Stores analytic data (if enabled)
// -- TYPES --
// 1. "login"
// 2. "tweets viewed"
// 3. "tweets posted"
func StoreAnalyticData(data AnalyticData) {
	if !cfg.TrackAnalytics {
		return
	}

	inRamAnalytics = append(inRamAnalytics, data)
}

func WriteAnalyticsToDB() {
	if !cfg.TrackAnalytics {
		return
	}

	if len(inRamAnalytics) == 0 {
		return
	}

	tx := db.Begin()

	if err := tx.CreateInBatches(inRamAnalytics, 100).Error; err != nil {
		tx.Rollback()
		log.Printf("Error writing analytics data: %v", err)
		return
	}

	if err := tx.Commit().Error; err != nil {
		log.Printf("Error committing analytics transaction: %v", err)
		return
	}

	log.Printf("Successfully wrote %d analytics records to database", len(inRamAnalytics))
	inRamAnalytics = inRamAnalytics[:0]
}

func StartPeriodicAnalyticsWriter(interval time.Duration) {
	if !cfg.TrackAnalytics {
		return
	}

	analyticsTicker = time.NewTicker(interval)

	go func() {
		for range analyticsTicker.C {
			WriteAnalyticsToDB()
		}
	}()
}
