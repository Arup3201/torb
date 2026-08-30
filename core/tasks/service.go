package tasks

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/core/assignees"
	"github.com/Arup3201/torb/core/documents"
	"github.com/Arup3201/torb/core/members"
)

type ProjectTaskItem struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description *string   `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	ProjectID string               `json:"project_id"`
	Assignees []assignees.Assignee `json:"assignees"`
}

type DashboardTaskItem struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description *string   `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	ProjectID   string `json:"project_id"`
	ProjectName string `json:"project_name"`
}

type TaskService struct {
	taskRepo     *TaskRepository
	memberRepo   *members.MemberRepository
	assigneeRepo *assignees.AssigneeRepository
	docStorage   *documents.DocumentStorage
}

func NewTaskService(taskRepo *TaskRepository,
	memberRepo *members.MemberRepository,
	assigneeRepo *assignees.AssigneeRepository,
	docStorage *documents.DocumentStorage) *TaskService {
	return &TaskService{
		taskRepo:     taskRepo,
		memberRepo:   memberRepo,
		assigneeRepo: assigneeRepo,
		docStorage:   docStorage,
	}
}

func (s *TaskService) Create(ctx context.Context,
	projectID, userID string,
	title, description, status string) (string, error) {

	var err error

	err = core.NeedsToBeAnOwner(ctx, s.memberRepo, projectID, userID)
	if err != nil {
		return "", fmt.Errorf("needs to be an owner: %w", err)
	}

	if strings.Trim(title, " ") == "" {
		return "", core.ErrInvalidValue
	}

	if !slices.Contains([]string{
		core.TASK_STATUS_UNASSIGNED,
		core.TASK_STATUS_ONGOING,
		core.TASK_STATUS_COMPLETED,
		core.TASK_STATUS_ABANDONED,
	}, status) {
		return "", core.ErrInvalidValue
	}

	taskID, err := s.taskRepo.Create(ctx,
		projectID,
		title, description, status)
	if err != nil {
		return "", fmt.Errorf("task repository create: %w", err)
	}

	return taskID, nil
}

func (s *TaskService) List(ctx context.Context,
	projectID, userID string) ([]ProjectTaskItem, error) {

	var err error

	err = core.NeedsToBeAMember(ctx, s.memberRepo, projectID, userID)
	if err != nil {
		return nil, fmt.Errorf("needs to be a member: %w", err)
	}

	rows, err := s.taskRepo.List(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("task repository list: %w", err)
	}

	tasks := []ProjectTaskItem{}
	var task ProjectTaskItem
	for _, r := range rows {
		task = ProjectTaskItem{
			ID:          r.ID,
			Title:       r.Title,
			Description: r.Description,
			Status:      r.Status.String,
			CreatedAt:   r.CreatedAt,
			UpdatedAt:   r.UpdatedAt,
			Assignees:   []assignees.Assignee{},
		}
		for _, assignee := range r.Assignees {
			url, _ := s.docStorage.GetObjectURL(ctx, assignee.AssigneeAvatarKey, 15*time.Minute)
			task.Assignees = append(task.Assignees, assignees.Assignee{
				ProjectID: assignee.ProjectID,
				TaskID:    assignee.TaskID,
				CreatedAt: assignee.CreatedAt,
				UpdatedAt: assignee.UpdatedAt,
				Avatar: core.Avatar{
					UserID:      assignee.AssigneeID,
					Username:    assignee.AssigneeUsername,
					DisplayName: assignee.AssigneeDisplayName,
					Email:       assignee.AssigneeEmail,
					AvatarURL:   &url,
				},
			})
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (s *TaskService) Get(ctx context.Context,
	projectID, taskID, userID string) (*ProjectTaskItem, error) {

	var err error

	err = core.NeedsToBeAMember(ctx, s.memberRepo, projectID, userID)
	if err != nil {
		return nil, fmt.Errorf("needs to be a member: %w", err)
	}

	row, err := s.taskRepo.Get(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("task repository get: %w", err)
	}

	task := ProjectTaskItem{
		ID:          row.ID,
		Title:       row.Title,
		Description: row.Description,
		Status:      row.Status.String,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
		Assignees:   []assignees.Assignee{},
	}
	for _, assignee := range row.Assignees {
		url, _ := s.docStorage.GetObjectURL(ctx, assignee.AssigneeAvatarKey, 15*time.Minute)
		task.Assignees = append(task.Assignees, assignees.Assignee{
			ProjectID: assignee.ProjectID,
			TaskID:    assignee.TaskID,
			CreatedAt: assignee.CreatedAt,
			UpdatedAt: assignee.UpdatedAt,
			Avatar: core.Avatar{
				UserID:      assignee.AssigneeID,
				Username:    assignee.AssigneeUsername,
				DisplayName: assignee.AssigneeDisplayName,
				Email:       assignee.AssigneeEmail,
				AvatarURL:   &url,
			},
		})
	}

	return &task, nil
}

func (s *TaskService) Update(ctx context.Context,
	projectID, taskID, userID string,
	title, description, status *string) error {

	var err error

	err = core.NeedsToBeAnOwner(ctx, s.memberRepo, projectID, userID)
	if err != nil {
		err = core.NeedsToBeAnAssignee(ctx, s.assigneeRepo, projectID, taskID, userID)
		if err != nil {
			return fmt.Errorf("needs to be an owner or assignee: %w", err)
		}
	}

	if title == nil && description == nil && status == nil {
		return core.ErrInvalidValue
	}

	if title != nil {
		err = s.taskRepo.Update(ctx, taskID, title, nil, nil)
		if err != nil {
			return fmt.Errorf("task repository update title: %w", err)
		}
	}

	if description != nil {
		err = s.taskRepo.Update(ctx, taskID, nil, description, nil)
		if err != nil {
			return fmt.Errorf("task repository update description: %w", err)
		}
	}

	if status != nil {
		if !slices.Contains([]string{
			core.TASK_STATUS_UNASSIGNED,
			core.TASK_STATUS_ONGOING,
			core.TASK_STATUS_COMPLETED,
			core.TASK_STATUS_ABANDONED,
		}, *status) {
			return core.ErrInvalidValue
		}

		err = s.taskRepo.Update(ctx, taskID, nil, nil, status)
		if err != nil {
			return fmt.Errorf("task repository update status: %w", err)
		}
	}

	return nil
}

func (s *TaskService) RecentlyAssigned(ctx context.Context,
	userId string) ([]DashboardTaskItem, error) {

	// pick last 10 recently joined projects in descending order of their joining time
	rows, err := s.taskRepo.RecentlyAssigned(ctx, userId, 10)
	if err != nil {
		return nil, fmt.Errorf("task repository recently assigned: %w", err)
	}

	tasks := []DashboardTaskItem{}
	for _, r := range rows {
		tasks = append(tasks, DashboardTaskItem{
			ID:          r.ID,
			Title:       r.Title,
			Description: r.Description,
			Status:      r.Status.String,
			CreatedAt:   r.CreatedAt,
			UpdatedAt:   r.UpdatedAt,
			ProjectID:   r.ProjectID,
			ProjectName: r.ProjectName,
		})
	}

	return tasks, nil
}

func (s *TaskService) RecentlyUnassigned(ctx context.Context,
	userId string) ([]DashboardTaskItem, error) {

	// pick last 10 recently joined projects in descending order of their joining time
	rows, err := s.taskRepo.RecentlyUnassigned(ctx, userId, 10)
	if err != nil {
		return nil, fmt.Errorf("task repository recently unassigned: %w", err)
	}

	tasks := []DashboardTaskItem{}
	for _, r := range rows {
		tasks = append(tasks, DashboardTaskItem{
			ID:          r.ID,
			Title:       r.Title,
			Description: r.Description,
			Status:      r.Status.String,
			CreatedAt:   r.CreatedAt,
			UpdatedAt:   r.UpdatedAt,
			ProjectID:   r.ProjectID,
			ProjectName: r.ProjectName,
		})
	}

	return tasks, nil
}
