package openid

import (
	"context"
	"errors"
	"fmt"

	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/models"
	"gorm.io/gorm"
)

type OauthRepository struct {
	db *gorm.DB
}

func NewOauthRepository(db *gorm.DB) *OauthRepository {
	return &OauthRepository{
		db: db,
	}
}

func (r *OauthRepository) WithTx(tx *gorm.DB) *OauthRepository {
	return NewOauthRepository(tx)
}

func (r *OauthRepository) Create(ctx context.Context,
	subject, provider, userID, email string) error {

	acc := models.OauthAccount{
		Subject:  subject,
		Provider: provider,
		UserID:   userID,
		Email:    email,
	}
	err := gorm.G[models.OauthAccount](r.db).Create(ctx, &acc)
	if err != nil {
		return fmt.Errorf("gorm create: %w", err)
	}

	return nil
}

func (r *OauthRepository) Get(ctx context.Context,
	subject, provider string) (models.OauthAccount, error) {

	acc, err := gorm.G[models.OauthAccount](r.db).Where("subject = ? AND provider = ?",
		subject, provider).First(ctx)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return acc, core.ErrNotFound
	} else if err != nil {
		return acc, fmt.Errorf("gorm query: %w", err)
	}

	return acc, nil
}
