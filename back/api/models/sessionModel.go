package models

import "gorm.io/gorm"

type Session struct {
	gorm.Model
	Name         string        `gorm:"unique"`
	HostID       uint          // User who created the session (host).
	Host         User          `gorm:"foreignKey:HostID"`
	Status       string        `gorm:"default:'active'"`     // Can be 'active' or 'ended'.
	UserSessions []UserSession `gorm:"foreignKey:SessionID"` // Relationship with user sessions.
}

type SessionStream struct {
	gorm.Model
	SessionID    uint    // Reference to the session.
	StreamFormat string  // e.g., "mp4" or "mpegdash"
	Status       string  `gorm:"default:'active'"` // Stream status: active, ended
	StartedAt    uint    `gorm:"autoCreateTime"`   // Timestamp when the stream starts.
	EndedAt      uint    `gorm:"default:NULL"`     // Timestamp when the stream ends.
	Session      Session `gorm:"foreignKey:SessionID"`
}
