package models

import "gorm.io/gorm"

type Session struct {
	gorm.Model
	Name         string        `gorm:"unique"`
	HostID       uint          // User who created the session (host).
	Host         User          `gorm:"foreignKey:HostID"`
	Status       string        `gorm:"default:'active'"`     // Can be 'active' or 'ended'.
	UserSessions []UserSession `gorm:"foreignKey:SessionID"` // Relationship with user sessions.
	McAddr       string        //Multicast address
}

type UserSession struct {
	gorm.Model
	UserID    uint
	SessionID uint
	JoinedAt  uint    `gorm:"autoCreateTime"` // The time when the user joined the session.
	LeftAt    uint    `gorm:"default:NULL"`   // The time when the user leaves the session.
	User      User    `gorm:"foreignKey:UserID"`
	Session   Session `gorm:"foreignKey:SessionID"`
}
