package comments

import (
	"context"
	"fmt"
	"time"

	"github.com/Arup3201/torb/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CommentRow struct {
	ID        string    `gorm:"column:id"`
	ProjectID string    `gorm:"column:project_id"`
	TaskID    string    `gorm:"column:task_id"`
	Content   string    `gorm:"column:content"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`

	UserID      string  `gorm:"column:user_id"`
	Username    string  `gorm:"column:username"`
	DisplayName *string `gorm:"column:display_name"`
	Email       string  `gorm:"column:email"`
	AvatarURL   *string `gorm:"column:avatar_url"`
}

type CommentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{
		db: db,
	}
}

func (r *CommentRepository) Create(ctx context.Context,
	projectId, taskId, userId string,
	content string) (string, error) {

	var err error

	id := uuid.NewString()
	comment := models.Comment{
		ID:        id,
		ProjectID: projectId,
		TaskID:    taskId,
		UserID:    userId,
		Content:   content,
	}

	err = gorm.G[models.Comment](r.db).Create(ctx, &comment)
	if err != nil {
		return "", fmt.Errorf("gorm create: %w", err)
	}

	return comment.ID, nil
}

func (r *CommentRepository) List(ctx context.Context,
	projectId, taskId string) ([]CommentRow, error) {

	var rows = []CommentRow{}
	err := r.db.WithContext(ctx).
		Table("comments c").
		Select(`c.id, c.project_id, c.task_id, c.content, 
				c.created_at, c.updated_at, 
				u.id as user_id, 
				u.username as username, 
				u.display_name as display_name, 
				u.email as email, 
				u.avatar_url as avatar_url`).
		Joins("INNER JOIN users as u ON u.id=c.user_id").
		Where("c.project_id = ? AND c.task_id = ?", projectId, taskId).
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("db query context: %w", err)
	}

	return rows, nil
}
