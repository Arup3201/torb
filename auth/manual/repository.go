package manual

import (
	"context"
	"fmt"
	"time"

	"github.com/Arup3201/torb/models"
	"gorm.io/gorm"
)

type ManualAccountRepository struct {
	db *gorm.DB
}

func NewManualAccountRepository(db *gorm.DB) *ManualAccountRepository {
	return &ManualAccountRepository{
		db: db,
	}
}

func (r *ManualAccountRepository) WithTx(tx *gorm.DB) *ManualAccountRepository {
	return NewManualAccountRepository(tx)
}

func (r *ManualAccountRepository) Create(ctx context.Context,
	userID, email string,
	passwordHash []byte,
	emailVerified bool) error {

	manualAccount := models.ManualAccount{
		UserID:        userID,
		Email:         email,
		PasswordHash:  passwordHash,
		EmailVerified: emailVerified,
	}
	err := gorm.G[models.ManualAccount](r.db).Create(ctx, &manualAccount)
	if err != nil {
		return fmt.Errorf("gorm create: %w", err)
	}

	return nil
}

func (r *ManualAccountRepository) Get(ctx context.Context,
	userID string) (models.ManualAccount, error) {

	return gorm.G[models.ManualAccount](r.db).
		Where("user_id = ?", userID).
		First(ctx)
}

func (r *ManualAccountRepository) GetByEmail(ctx context.Context,
	email string) (models.ManualAccount, error) {

	return gorm.G[models.ManualAccount](r.db).
		Where("email = ?", email).
		First(ctx)
}

func (r *ManualAccountRepository) GetByVerificationToken(ctx context.Context,
	token string) (models.ManualAccount, error) {

	return gorm.G[models.ManualAccount](r.db).
		Where("verification_token = ?", token).
		First(ctx)
}

func (r *ManualAccountRepository) GetByResetToken(ctx context.Context,
	token string) (models.ManualAccount, error) {

	return gorm.G[models.ManualAccount](r.db).
		Where("reset_password_token = ?", token).
		First(ctx)
}

func (r *ManualAccountRepository) UpdateVerificationToken(ctx context.Context,
	account models.ManualAccount,
	tokenHash string,
	tokenExpiresAt time.Time) error {

	account.VerificationToken = &tokenHash
	account.VerificationTokenExpiresAt = &tokenExpiresAt
	if err := r.db.Save(&account).Error; err != nil {
		return fmt.Errorf("gorm db save: %w", err)
	}

	return nil
}

func (r *ManualAccountRepository) UpdateResetPasswordToken(ctx context.Context,
	account models.ManualAccount,
	tokenHash string,
	tokenExpiresAt time.Time) error {

	account.ResetPasswordToken = &tokenHash
	account.ResetPasswordTokenExpiresAt = &tokenExpiresAt
	account.ResetPasswordTokenUsedAt = nil
	if err := r.db.Save(&account).Error; err != nil {
		return fmt.Errorf("gorm db save: %w", err)
	}

	return nil
}

func (r *ManualAccountRepository) UpdateEmailVerified(ctx context.Context,
	account models.ManualAccount,
	emailVerified bool) error {

	account.EmailVerified = emailVerified
	if err := r.db.Save(&account).Error; err != nil {
		return fmt.Errorf("gorm db save: %w", err)
	}

	return nil
}

func (r *ManualAccountRepository) UpdatePassword(ctx context.Context,
	account models.ManualAccount,
	passwordHash []byte) error {

	account.PasswordHash = passwordHash
	if err := r.db.Save(&account).Error; err != nil {
		return fmt.Errorf("gorm db save: %w", err)
	}

	return nil
}
