package models

import (
	"time"
)

type Assignee struct {
	ProjectID string `gorm:"primaryKey"`
	TaskID    string `gorm:"primaryKey"`
	UserID    string `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
