package members

import (
	"context"
	"fmt"
	"time"

	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/models"
	"gorm.io/gorm"
)

type MemberRow struct {
	ProjectID string          `gorm:"column:project_id"`
	Role      models.UserRole `gorm:"column:role"`
	CreatedAt time.Time       `gorm:"column:created_at"`
	UpdatedAt time.Time       `gorm:"column:updated_at"`

	UserID      string  `gorm:"column:member_user_id"`
	Username    string  `gorm:"column:member_username"`
	DisplayName *string `gorm:"column:member_display_name"`
	Email       string  `gorm:"column:member_email"`
	AvatarKey   *string `gorm:"column:member_avatar_key"`
}

type MemberRepository struct {
	db *gorm.DB
}

func NewMemberRepository(db *gorm.DB) *MemberRepository {
	return &MemberRepository{
		db: db,
	}
}

func (r *MemberRepository) WithTx(tx *gorm.DB) *MemberRepository {
	return NewMemberRepository(tx)
}

func (r *MemberRepository) Create(ctx context.Context,
	projectID, userID, userRole string) error {

	role := models.Member{
		ProjectID: projectID,
		UserID:    userID,
		Role: models.UserRole{
			String: userRole,
		},
	}
	err := gorm.G[models.Member](r.db).Create(ctx, &role)
	if err != nil {
		return fmt.Errorf("gorm create: %w", err)
	}

	return nil
}

func (r *MemberRepository) Role(ctx context.Context,
	projectID, userID string) (string, error) {

	member, err := gorm.G[models.Member](r.db).Where(
		"project_id = ? AND user_id = ?",
		projectID, userID).First(ctx)
	if err == gorm.ErrRecordNotFound {
		return "", core.ErrNotFound
	} else if err != nil {
		return "", fmt.Errorf("gorm query: %w", err)
	}

	return member.Role.String, nil
}

func (r *MemberRepository) Count(ctx context.Context,
	projectID string) (int64, error) {

	var cnt int64
	err := r.db.Model(&models.Member{}).Where("project_id = ?", projectID).Count(&cnt).Error
	if err != nil {
		return -1, fmt.Errorf("gorm db count query: %w", err)
	}

	return cnt, nil
}

func (r *MemberRepository) List(ctx context.Context,
	projectID string) ([]MemberRow, error) {

	var rows []MemberRow
	err := r.db.WithContext(ctx).
		Table("members m").
		Select(`m.project_id, m.role, m.created_at, m.updated_at, 
				u.id as member_user_id, 
				u.username as member_username, 
				u.display_name as member_display_name, 
				u.email as member_email, 
				u.avatar_key as member_avatar_key`).
		Joins("INNER JOIN users AS u ON m.user_id=u.id").
		Where("m.project_id = ?", projectID).
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("gorm db scan: %w", err)
	}

	return rows, nil
}
