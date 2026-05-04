package assignees

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/core/members"
)

type Assignee struct {
	ProjectID   string    `json:"project_id"`
	TaskID      string    `json:"task_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	core.Avatar `json:"avatar"`
}

type AssigneeService struct {
	memberRepo   *members.MemberRepository
	assigneeRepo *AssigneeRepository
}

func NewAssigneeService(memberRepo *members.MemberRepository,
	assigneeRepo *AssigneeRepository) *AssigneeService {
	return &AssigneeService{
		memberRepo:   memberRepo,
		assigneeRepo: assigneeRepo,
	}
}

func (s *AssigneeService) AddAssignee(ctx context.Context,
	projectID, taskID, userID, assigneeID string) error {

	var err error

	err = core.NeedsToBeAnOwner(ctx, s.memberRepo, projectID, userID)
	if err != nil {
		return fmt.Errorf("needs to be a owner: %w", err)
	}

	if err = s.assigneeRepo.Is(ctx, projectID, taskID, assigneeID); err == nil {
		return core.ErrDuplicate
	}

	err = s.assigneeRepo.Create(ctx, projectID, taskID, assigneeID)
	if err != nil {
		return fmt.Errorf("assignee repository create: %w", err)
	}

	return nil
}

func (s *AssigneeService) RemoveAssignee(ctx context.Context,
	projectID, taskID, userID, assigneeID string) error {

	var err error

	err = core.NeedsToBeAnOwner(ctx, s.memberRepo, projectID, userID)
	if err != nil {
		return fmt.Errorf("needs to be a owner: %w", err)
	}

	if err = s.assigneeRepo.Is(ctx, projectID, taskID, assigneeID); errors.Is(err, core.ErrNotFound) {
		return core.ErrNotFound
	}

	err = s.assigneeRepo.Delete(ctx, projectID, taskID, assigneeID)
	if err != nil {
		return fmt.Errorf("assignee repository create: %w", err)
	}

	return nil
}
