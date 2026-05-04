package models

import "time"

type OauthAccount struct {
	Subject   string `gorm:"index:idx_provider_subject,unique"`
	Provider  string `gorm:"index:idx_provider_subject,unique;index:idx_provider_user,unique"`
	UserID    string `gorm:"index:idx_provider_user,unique"`
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
}
