package tasks

import (
	"context"
	"fmt"
	"time"

	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/models"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

/*
Used with datatypes.JSONSlice to store list of AssigneeRow

It uses json unmarshalling to map the values from query result.
So json tags are used instead of gorm tags.
*/
type AssigneeRow struct {
	ProjectID string    `json:"project_id"`
	TaskID    string    `json:"task_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	AssigneeID          string  `json:"assignee_id"`
	AssigneeUsername    string  `json:"assignee_username"`
	AssigneeDisplayName *string `json:"assignee_display_name"`
	AssigneeEmail       string  `json:"assignee_email"`
	AssigneeAvatarKey   *string `json:"assignee_avatar_key"`
}

type ProjectTaskItemRow struct {
	ID          string            `gorm:"column:id"`
	Title       string            `gorm:"column:title"`
	Description *string           `gorm:"column:description"`
	Status      models.TaskStatus `gorm:"column:status"`
	CreatedAt   time.Time         `gorm:"column:created_at"`
	UpdatedAt   time.Time         `gorm:"column:updated_at"`

	ProjectID string                           `gorm:"column:project_id"`
	Assignees datatypes.JSONSlice[AssigneeRow] `gorm:"column:assignees"`
}

type DashboardTaskItemRow struct {
	ID          string            `gorm:"column:id"`
	Title       string            `gorm:"column:title"`
	Description *string           `gorm:"column:description"`
	Status      models.TaskStatus `gorm:"column:status"`
	CreatedAt   time.Time         `gorm:"column:created_at"`
	UpdatedAt   time.Time         `gorm:"column:updated_at"`

	ProjectID   string `gorm:"column:project_id"`
	ProjectName string `gorm:"column:project_name"`
}

type TaskRepository struct {
	db *gorm.DB
}

func NewTaskRepository(db *gorm.DB) *TaskRepository {
	return &TaskRepository{
		db: db,
	}
}

func (r *TaskRepository) Create(ctx context.Context,
	projectID string,
	title, description, status string) (string, error) {

	id := uuid.NewString()
	task := models.Task{
		ID:          id,
		ProjectID:   projectID,
		Title:       title,
		Description: &description,
		Status: models.TaskStatus{
			String: status,
		},
	}

	err := gorm.G[models.Task](r.db).Create(ctx, &task)
	if err != nil {
		return "", fmt.Errorf("gorm create: %w", err)
	}

	return task.ID, nil
}

func (r *TaskRepository) Get(ctx context.Context, id string) (ProjectTaskItemRow, error) {

	query := `SELECT 
			t.id, t.title, t.description, t.status, t.created_at, t.updated_at, 
			COALESCE(
				json_agg(
				json_build_object(
					'project_id', a.project_id,
					'task_id', a.task_id,
					'assignee_id', a.user_id,
					'assignee_username', u.username,
					'assignee_display_name', u.display_name,
					'assignee_email', u.email,
					'assignee_avatar_key', u.avatar_key
				)
				) FILTER (WHERE a.user_id IS NOT NULL),
				'[]'
			) AS assignees 
			FROM tasks AS t 
			LEFT JOIN assignees AS a ON a.task_id=t.id 
			LEFT JOIN users AS u ON u.id=a.user_id 
			WHERE t.id = ? 
			GROUP BY t.id, t.title, t.status, t.created_at, t.updated_at`

	task, err := gorm.G[ProjectTaskItemRow](r.db).Raw(query, id).First(ctx)
	if err != nil {
		return task, fmt.Errorf("gorm query task: %w", err)
	}

	return task, nil
}

func (r *TaskRepository) List(ctx context.Context,
	projectId string) ([]ProjectTaskItemRow, error) {

	query := `SELECT 
		t.id, t.title, t.status, t.created_at, t.updated_at, 
		COALESCE(
			json_agg(
			json_build_object(
				'project_id', a.project_id,
				'task_id', a.task_id,
				'assignee_id', a.user_id,
				'assignee_username', u.username,
				'assignee_display_name', u.display_name,
				'assignee_email', u.email,
				'assignee_avatar_key', u.avatar_key
			)
			) FILTER (WHERE a.user_id IS NOT NULL),
			'[]'
		) AS assignees 
		FROM tasks AS t 
		LEFT JOIN assignees AS a ON a.task_id=t.id 
		LEFT JOIN users AS u ON u.id=a.user_id 
		WHERE t.project_id=? 
		GROUP BY t.id`

	var rows = []ProjectTaskItemRow{}
	err := r.db.Raw(query, projectId).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("gorm db raw scan: %w", err)
	}

	return rows, nil
}

func (r *TaskRepository) Update(ctx context.Context, id string,
	title, description, status *string) error {

	task, err := gorm.G[models.Task](r.db).Where("id = ?", id).First(ctx)
	if err != nil {
		return fmt.Errorf("gorm query task: %w", err)
	}

	if title != nil {
		task.Title = *title
	}

	if description != nil {
		*task.Description = *description
	}

	if status != nil {
		task.Status = models.TaskStatus{
			String: *status,
		}
	}

	err = r.db.Save(&task).Error
	if err != nil {
		return fmt.Errorf("gorm db save: %w", err)
	}

	return nil
}

func (r *TaskRepository) RecentlyAssigned(ctx context.Context,
	userId string,
	n int) ([]DashboardTaskItemRow, error) {

	var rows = []DashboardTaskItemRow{}
	err := r.db.WithContext(ctx).
		Table("tasks t").
		Select(`t.id, t.title, t.status, t.created_at, t.updated_at,
				p.id as project_id, p.name as project_name`).
		Joins("INNER JOIN projects AS p ON t.project_id=p.id").
		Joins("INNER JOIN assignees AS a ON a.task_id=t.id").
		Where("a.user_id = ?", userId).
		Order("a.created_at DESC").
		Limit(n).
		Scan(&rows).
		Error
	if err != nil {
		return nil, fmt.Errorf("gorm db scan: %w", err)
	}

	return rows, nil
}

func (r *TaskRepository) RecentlyUnassigned(ctx context.Context,
	userId string,
	n int) ([]DashboardTaskItemRow, error) {

	var rows = []DashboardTaskItemRow{}
	err := r.db.WithContext(ctx).
		Table("tasks t").
		Select(`t.id, t.project_id, t.title, t.status, t.created_at, t.updated_at,
					p.name as project_name`).
		Joins("INNER JOIN projects AS p ON t.project_id=p.id").
		Where("p.owner_id = ? AND t.status = ?", userId, core.TASK_STATUS_UNASSIGNED).
		Order("t.created_at DESC").
		Limit(n).
		Scan(&rows).
		Error
	if err != nil {
		return nil, fmt.Errorf("gorm db scan: %w", err)
	}

	return rows, nil
}
