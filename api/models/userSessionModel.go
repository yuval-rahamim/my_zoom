package models

import "gorm.io/gorm"

type UserSession struct {
	gorm.Model
	UserID    uint
	SessionID uint
	JoinedAt  uint    `gorm:"autoCreateTime"` // The time when the user joined the session.
	LeftAt    uint    `gorm:"default:NULL"`   // The time when the user leaves the session.
	User      User    `gorm:"foreignKey:UserID"`
	Session   Session `gorm:"foreignKey:SessionID"`
}
