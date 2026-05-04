package assignees

import (
	"context"
	"fmt"

	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/models"
	"gorm.io/gorm"
)

type AssigneeRepository struct {
	db *gorm.DB
}

func NewAssigneeRepository(db *gorm.DB) *AssigneeRepository {
	return &AssigneeRepository{
		db: db,
	}
}

func (r *AssigneeRepository) Create(ctx context.Context,
	projectID, taskID, userID string) error {

	var err error

	assignee := models.Assignee{
		ProjectID: projectID,
		TaskID:    taskID,
		UserID:    userID,
	}
	err = gorm.G[models.Assignee](r.db).Create(ctx, &assignee)
	if err != nil {
		return fmt.Errorf("gorm create: %w", err)
	}

	return nil
}

func (r *AssigneeRepository) Is(ctx context.Context,
	projectID, taskID, userID string) error {

	_, err := gorm.G[models.Assignee](r.db).Where(
		"project_id = ? AND task_id = ? AND user_id = ?",
		projectID, taskID, userID).First(ctx)
	if err == gorm.ErrRecordNotFound {
		return core.ErrNotFound
	} else if err != nil {
		return fmt.Errorf("gorm query: %w", err)
	}

	return nil
}

func (r *AssigneeRepository) Delete(ctx context.Context,
	projectID, taskID, userID string) error {

	_, err := gorm.G[models.Assignee](r.db).Where(
		"project_id = ? AND task_id = ? AND user_id = ?",
		projectID, taskID, userID).Delete(ctx)
	if err != nil {
		return fmt.Errorf("gorm query delete: %w", err)
	}

	return nil
}
