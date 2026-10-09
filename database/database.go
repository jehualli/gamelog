// Package database manages the SQLite connection and auto-migration.
// LEARN: Auto-migration reads your model structs and creates/alters database tables
// to match. It adds missing columns and indexes but never drops existing ones —
// so it's safe to run on every startup.
package database

import (
	"log"

	"gamelog/models"

	"github.com/glebarez/sqlite" // pure-Go SQLite driver (no CGO required)
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB is the global database connection. Import this in any package that needs DB access.
var DB *gorm.DB

// Connect opens the SQLite file and runs migrations.
// For local learning, SQLite is perfect: one file, no server to run.
// In production you'd swap the driver for postgres or mysql.
func Connect(path string) {
	var err error

	// LEARN: gorm.Config lets you tune behavior. Here we set the log level to Warn
	// so you see slow queries and errors, but not every SELECT during development.
	DB, err = gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("database: failed to connect to %s: %v", path, err)
	}

	// LEARN: AutoMigrate compares your structs to the current schema and applies
	// the minimum changes needed (CREATE TABLE, ADD COLUMN, CREATE INDEX).
	// Pass all models that should have a table.
	if err := DB.AutoMigrate(&models.User{}, &models.Game{}, &models.JournalEntry{}); err != nil {
		log.Fatalf("database: migration failed: %v", err)
	}

	log.Printf("database: connected to %s (tables migrated)", path)
}
