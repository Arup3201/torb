package models

import (
	"time"
)

type Comment struct {
	ID        string `gorm:"primaryKey"`
	ProjectID string `gorm:"index:idx_project_task_comment"`
	TaskID    string `gorm:"index:idx_project_task_comment"`
	UserID    string
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
}
