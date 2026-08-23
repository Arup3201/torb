package models

import "time"

type S3Document struct {
	ID          string `gorm:"primaryKey"`
	Key         string `gorm:"unique"`
	Type        string
	ContentType string
	Size        uint
	CreatedAt   time.Time
	DeletedAt   *time.Time
}
