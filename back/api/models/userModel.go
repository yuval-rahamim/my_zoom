package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Name      string `gorm:"unique"`
	Password  []byte `json:"-"`
	ImgPath   string
	isManager bool
}
