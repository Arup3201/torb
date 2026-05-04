package models

import "time"

type ManualAccount struct {
	UserID       string `gorm:"primaryKey"`
	Email        string `gorm:"unique"`
	PasswordHash []byte

	EmailVerified              bool
	VerificationToken          *string
	VerificationTokenExpiresAt *time.Time

	ResetPasswordToken          *string
	ResetPasswordTokenExpiresAt *time.Time
	ResetPasswordTokenUsedAt    *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}
