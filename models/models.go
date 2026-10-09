// Package models defines the database schema using GORM structs.
// LEARN: GORM is an ORM (Object-Relational Mapper). Instead of writing SQL by hand,
// you define Go structs and GORM translates them to database tables.
// Struct tags like `gorm:"not null"` add constraints at the database level.
package models

import (
	"time"

	"gorm.io/gorm"
)

// User is a registered account. Passwords are stored as bcrypt hashes — never plain text.
// LEARN: The `gorm:"uniqueIndex"` tag creates a database unique constraint AND an index
// (fast lookup). Without an index, finding a user by email would scan every row.
type User struct {
	ID        uint           `gorm:"primarykey"`
	Username  string         `gorm:"uniqueIndex;not null;size:50"`
	Email     string         `gorm:"uniqueIndex;not null;size:255"`
	Password  string         `gorm:"not null"` // bcrypt hash
	Role      string         `gorm:"not null;default:player"` // "player" or "admin"
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"` // soft delete: sets deleted_at instead of removing row
	Games     []Game         `gorm:"foreignKey:UserID"`
}

// Game tracks a video game the user is playing.
// LEARN: The relationship `Games []Game` + `gorm:"foreignKey:UserID"` tells GORM
// that User "has many" Games. When you do db.Preload("Games").Find(&user), GORM
// runs a second query to load all associated games automatically.
type Game struct {
	ID        uint           `gorm:"primarykey"`
	UserID    uint           `gorm:"not null;index"` // index speeds up "get all games for user X"
	Title     string         `gorm:"not null;size:255"`
	Platform  string         `gorm:"size:100"` // PC, PS5, Switch, etc.
	Genre     string         `gorm:"size:100"`
	Status    string         `gorm:"not null;default:playing"` // playing|completed|paused|dropped|backlog
	Rating    int            `gorm:"default:0"`                 // 0 = unrated, 1–10
	CoverURL  string         `gorm:"size:512"`                  // Cloudinary image URL
	Notes     string         `gorm:"type:text"`                 // general notes
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	Entries   []JournalEntry `gorm:"foreignKey:GameID"`
}

// JournalEntry is a session log — one entry per play session or milestone.
// This is the heart of the tracker: stage, thoughts, blockers, feelings.
type JournalEntry struct {
	ID         uint      `gorm:"primarykey"`
	GameID     uint      `gorm:"not null;index"`
	UserID     uint      `gorm:"not null;index"`
	Stage      string    `gorm:"size:255"` // current level / boss / chapter
	Notes      string    `gorm:"type:text"` // observations, strategies, questions
	Impediment string    `gorm:"type:text"` // what's blocking you
	Emotion    string    `gorm:"size:100"`  // excited, frustrated, focused, etc.
	ImageURL   string    `gorm:"size:512"`  // Cloudinary screenshot URL
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
