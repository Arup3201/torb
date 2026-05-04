package requests

import (
	"context"
	"fmt"
	"time"

	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/models"
	"gorm.io/gorm"
)

type JoinRequestRow struct {
	ProjectID string            `gorm:"column:project_id"`
	Status    models.JoinStatus `gorm:"column:status"`
	CreatedAt time.Time         `gorm:"column:created_at"`
	UpdatedAt time.Time         `gorm:"column:updated_at"`

	UserID      string  `gorm:"column:requestor_user_id"`
	Username    string  `gorm:"column:requestor_username"`
	DisplayName *string `gorm:"column:requestor_display_name"`
	Email       string  `gorm:"column:requestor_email"`
	AvatarURL   *string `gorm:"column:requestor_avatar_url"`
}

type JoinRepository struct {
	db *gorm.DB
}

func NewJoinRepository(db *gorm.DB) *JoinRepository {
	return &JoinRepository{db: db}
}

func (r *JoinRepository) WithTx(tx *gorm.DB) *JoinRepository {
	return NewJoinRepository(tx)
}

func (r *JoinRepository) Create(ctx context.Context,
	projectID, userID, joinStatus string) error {

	var err error

	joinReq := models.JoinRequest{
		ProjectID: projectID,
		UserID:    userID,
		Status: models.JoinStatus{
			String: joinStatus,
		},
	}
	err = gorm.G[models.JoinRequest](r.db).Create(ctx, &joinReq)
	if err == gorm.ErrDuplicatedKey {
		return core.ErrDuplicate
	} else if err != nil {
		return fmt.Errorf("gorm create: %w", err)
	}

	return nil
}

func (r *JoinRepository) List(ctx context.Context, projectID string) ([]JoinRequestRow, error) {

	var rows = []JoinRequestRow{}
	err := r.db.WithContext(ctx).
		Table("join_requests jr").
		Select(`jr.project_id, jr.status, jr.created_at, jr.updated_at, 
				u.id as requestor_user_id, 
				u.username as requestor_username, 
				u.display_name as requestor_diplay_name, 
				u.email as requestor_email, 
				u.avatar_url as requestor_avatar_url`).
		Joins("INNER JOIN users as u ON u.id=jr.user_id").
		Where("jr.project_id = ?", projectID).
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("db query context: %w", err)
	}

	return rows, nil
}

func (r *JoinRepository) Status(ctx context.Context, projectID, userID string) (string, error) {

	joinReq, err := gorm.G[models.JoinRequest](r.db).Where(
		"project_id = ? AND user_id = ?",
		projectID, userID).First(ctx)
	if err == gorm.ErrRecordNotFound {
		return "", core.ErrNotFound
	} else if err != nil {
		return "", fmt.Errorf("gorm query: %w", err)
	}

	return joinReq.Status.String, nil
}

func (r *JoinRepository) Update(ctx context.Context, projectID, userID, joinStatus string) error {

	joinReq, err := gorm.G[models.JoinRequest](r.db).Where(
		"project_id = ? AND user_id = ?",
		projectID, userID).First(ctx)
	if err == gorm.ErrRecordNotFound {
		return core.ErrNotFound
	} else if err != nil {
		return fmt.Errorf("join request get: %w", err)
	}

	joinReq.Status = models.JoinStatus{
		String: joinStatus,
	}
	err = r.db.Save(&joinReq).Error
	if err == gorm.ErrInvalidValue {
		return core.ErrInvalidValue
	} else if err != nil {
		return fmt.Errorf("gorm db save: %w", err)
	}

	return nil
}
