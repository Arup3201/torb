package users

import (
	"context"
	"fmt"
	"strings"

	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

func (r *UserRepository) WithTx(tx *gorm.DB) *UserRepository {
	return NewUserRepository(tx)
}

func (r *UserRepository) Create(ctx context.Context,
	username, email string,
	skills string,
	displayName, avatarUrl *string) (string, error) {

	id := uuid.NewString()
	user := models.User{
		ID:          id,
		Username:    username,
		Email:       email,
		DisplayName: displayName,
		AvatarURL:   avatarUrl,
		Skills:      skills,
	}

	err := gorm.G[models.User](r.db).Create(ctx, &user)

	if err != nil {
		if strings.Contains(err.Error(), `duplicate key value violates unique constraint`) {
			return "", core.ErrDuplicate
		}
		return "", fmt.Errorf("gorm create: %w", err)
	}

	return user.ID, nil
}

func (r *UserRepository) Get(ctx context.Context, id string) (models.User, error) {

	user, err := gorm.G[models.User](r.db).Where("id = ?", id).First(ctx)
	if err == gorm.ErrRecordNotFound {
		return user, core.ErrNotFound
	} else if err != nil {
		return user, fmt.Errorf("gorm query: %w", err)
	}

	return user, nil
}

func (r *UserRepository) CountProjects(ctx context.Context,
	id string) (int64, error) {

	var cnt int64
	err := r.db.WithContext(ctx).
		Table("members").
		Where("user_id = ?", id).
		Count(&cnt).
		Error
	if err != nil {
		return -1, err
	}

	return cnt, nil
}

func (r *UserRepository) CountTasks(ctx context.Context,
	id string) (int64, error) {

	var cnt int64
	err := r.db.WithContext(ctx).
		Table("tasks t").
		Joins("INNER JOIN assignees as a ON t.id=a.task_id").
		Where("a.user_id = ?", id).
		Count(&cnt).
		Error
	if err != nil {
		return -1, err
	}

	return cnt, nil
}

func (r *UserRepository) CountCompletedTasks(ctx context.Context,
	id string) (int64, error) {

	var cnt int64
	err := r.db.WithContext(ctx).
		Table("tasks t").
		Joins("INNER JOIN assignees as a ON t.id=a.task_id").
		Where("a.user_id = ? AND t.status = ?", id, core.TASK_STATUS_COMPLETED).
		Count(&cnt).
		Error
	if err != nil {
		return -1, err
	}

	return cnt, nil
}
