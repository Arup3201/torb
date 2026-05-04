package models

import "time"

type User struct {
	ID            string  `gorm:"primaryKey"`
	Username      string  `gorm:"index:idx_username,unique"`
	DisplayName   *string // nullable
	Email         string  `gorm:"unique"`
	AvatarURL     *string // nullable
	CreatedAt     time.Time
	UpdatedAt     time.Time
	LastLoginTime time.Time

	ManualAccounts []ManualAccount `gorm:"constraint:OnDelete:CASCADE"`
	OauthAccounts  []OauthAccount  `gorm:"constraint:OnDelete:CASCADE"`
	Projects       []Project       `gorm:"foreignKey:OwnerID;constraint:OnDelete:CASCADE"`
	Members        []Member        `gorm:"constraint:OnDelete:CASCADE"`
	JoinRequests   []JoinRequest   `gorm:"constraint:OnDelete:CASCADE"`
	Comments       []Comment       `gorm:"constraint:OnDelete:CASCADE"`
	Assignees      []Assignee      `gorm:"constraint:OnDelete:CASCADE"`
	Notifications  []Notification  `gorm:"constraint:OnDelete:CASCADE"`
}
