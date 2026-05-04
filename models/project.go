package models

import (
	"time"
)

type Project struct {
	ID          string `gorm:"primaryKey"`
	Name        string
	Description *string
	Skills      *string
	OwnerID     string
	CreatedAt   time.Time
	UpdatedAt   time.Time

	Tasks        []Task        `gorm:"constraint:OnDelete:CASCADE"`
	Roles        []Member      `gorm:"constraint:OnDelete:CASCADE"`
	JoinRequests []JoinRequest `gorm:"constraint:OnDelete:CASCADE"`
	Assignees    []Assignee    `gorm:"constraint:OnDelete:CASCADE"`
	Comments     []Comment     `gorm:"constraint:OnDelete:CASCADE"`
}
