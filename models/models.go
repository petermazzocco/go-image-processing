package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	Name      string         `gorm:"size:255;not null"`
	Email     string         `gorm:"size:255;not null;unique"`
	Images    []Image
}

type Image struct {
	ID        uint   `gorm:"primarykey"`
	UUID      string `gorm:"type:uuid;uniqueIndex"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	UserID    uint           `gorm:"not null"`
	User      *User          `json:"user,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Filename  string
	ImageData []byte
	R2Key     string
	MimeType  string
}
