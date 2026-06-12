package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/core/members"
	"github.com/Arup3201/torb/core/projects"
	"github.com/Arup3201/torb/core/tasks"
	"github.com/Arup3201/torb/core/users"
)

const (
	NT_TASK_ADDED       = "task_added"
	NT_TASK_UPDATED     = "task_updated"
	NT_ASSIGNEE_ADDED   = "assignee_added"
	NT_ASSIGNEE_REMOVED = "assignee_removed"
	NT_JOIN_REQUESTED   = "join_requested"
	NT_JOIN_RESPONDED   = "join_responded"
	NT_COMMENT_ADDED    = "comment_added"
)

type ProjectBody struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type TaskBody struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type TaskUpdateBody struct {
	To    string `json:"to"`
	Field string `json:"field"`
}

type TaskAdded struct {
	Project ProjectBody `json:"project"`
	Task    TaskBody    `json:"task"`
}

type TaskUpdated struct {
	Project ProjectBody      `json:"project"`
	Task    TaskBody         `json:"task"`
	Updates []TaskUpdateBody `json:"updates"`
	Updater core.Avatar      `json:"updater"`
}

type AssigneeUpdated struct {
	Project  ProjectBody `json:"project"`
	Task     TaskBody    `json:"task"`
	Assignee core.Avatar `json:"assignee"`
}

type JoinRequested struct {
	Project   ProjectBody `json:"project"`
	Requestor core.Avatar `json:"requestor"`
}

type JoinResponded struct {
	Project   ProjectBody `json:"project"`
	Responder core.Avatar `json:"responder"`
	Status    string      `json:"status"`
}

type CommentAdded struct {
	Project   ProjectBody `json:"project"`
	Task      TaskBody    `json:"task"`
	Commenter core.Avatar `json:"commenter"`
}

type Notification struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Type      string    `json:"type"`
	Body      any       `json:"body"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"created_at"`
}

type NotificationService struct {
	projectRepo      *projects.ProjectRepository
	taskRepo         *tasks.TaskRepository
	membershipRepo   *members.MemberRepository
	userRepo         *users.UserRepository
	notificationRepo *NotificationRepository
}

func NewNotificationService(
	projectRepo *projects.ProjectRepository,
	taskRepo *tasks.TaskRepository,
	membershipRepo *members.MemberRepository,
	userRepo *users.UserRepository,
	notificationRepo *NotificationRepository,
) *NotificationService {
	return &NotificationService{
		projectRepo:      projectRepo,
		taskRepo:         taskRepo,
		membershipRepo:   membershipRepo,
		userRepo:         userRepo,
		notificationRepo: notificationRepo,
	}
}

func (s *NotificationService) TaskAdded(ctx context.Context,
	projectID, taskID string) error {

	project, err := s.projectRepo.Get(ctx, projectID)
	if err != nil {
		return fmt.Errorf("project repository Get: %w", err)
	}

	task, err := s.taskRepo.Get(ctx, taskID)
	if err != nil {
		return fmt.Errorf("task repository Get: %w", err)
	}

	members, err := s.membershipRepo.List(ctx, projectID)
	if err != nil {
		return fmt.Errorf("membership repository Get: %w", err)
	}

	body, _ := json.Marshal(TaskAdded{
		Project: ProjectBody{
			ID:   projectID,
			Name: project.Name,
		},
		Task: TaskBody{
			ID:    taskID,
			Title: task.Title,
		},
	})
	for _, m := range members {
		if m.Role.String == core.ROLE_OWNER {
			continue
		}

		_, err := s.notificationRepo.Create(ctx, m.UserID, NT_TASK_ADDED, body, false)
		if err != nil {
			return fmt.Errorf("notification repository Create: %w", err)
		}
	}

	return nil
}

func (s *NotificationService) TaskUpdated(ctx context.Context,
	projectID, taskID string,
	title, description, status *string,
	updaterID string) error {

	project, err := s.projectRepo.Get(ctx, projectID)
	if err != nil {
		return fmt.Errorf("project repository Get: %w", err)
	}

	task, err := s.taskRepo.Get(ctx, taskID)
	if err != nil {
		return fmt.Errorf("task repository Get: %w", err)
	}

	updater, err := s.userRepo.Get(ctx, updaterID)
	if err != nil {
		return fmt.Errorf("user repository Get: %w", err)
	}

	updates := []TaskUpdateBody{}
	if title != nil {
		updates = append(updates, TaskUpdateBody{
			To:    *title,
			Field: "Title",
		})
	}
	if description != nil {
		updates = append(updates, TaskUpdateBody{
			To:    *description,
			Field: "Description",
		})
	}
	if status != nil {
		updates = append(updates, TaskUpdateBody{
			To:    *status,
			Field: "Status",
		})
	}

	body, _ := json.Marshal(TaskUpdated{
		Project: ProjectBody{
			ID:   projectID,
			Name: project.Name,
		},
		Task: TaskBody{
			ID:    taskID,
			Title: task.Title,
		},
		Updates: updates,
		Updater: core.Avatar{
			UserID:      updater.ID,
			Username:    updater.Username,
			DisplayName: updater.DisplayName,
			Email:       updater.Email,
			AvatarURL:   updater.AvatarURL,
		},
	})

	if project.OwnerID != updaterID {
		_, err := s.notificationRepo.Create(
			ctx,
			project.OwnerID,
			NT_TASK_UPDATED,
			body,
			false,
		)
		if err != nil {
			return fmt.Errorf("notification repository Create: %w", err)
		}
	}

	for _, assignee := range task.Assignees {
		if assignee.AssigneeID == updaterID {
			continue
		}

		_, err := s.notificationRepo.Create(
			ctx,
			assignee.AssigneeID,
			NT_TASK_UPDATED,
			body,
			false,
		)
		if err != nil {
			return fmt.Errorf("notification repository Create: %w", err)
		}
	}

	return nil
}

func (s *NotificationService) AssigneeUpdated(ctx context.Context,
	projectID, taskID string,
	assigneeID string,
	isAdded bool) error {

	project, err := s.projectRepo.Get(ctx, projectID)
	if err != nil {
		return fmt.Errorf("project repository Get: %w", err)
	}

	task, err := s.taskRepo.Get(ctx, taskID)
	if err != nil {
		return fmt.Errorf("task repository Get: %w", err)
	}

	assignee, err := s.userRepo.Get(ctx, assigneeID)
	if err != nil {
		return fmt.Errorf("user repository Get: %w", err)
	}

	body, _ := json.Marshal(AssigneeUpdated{
		Project: ProjectBody{
			ID:   projectID,
			Name: project.Name,
		},
		Task: TaskBody{
			ID:    taskID,
			Title: task.Title,
		},
		Assignee: core.Avatar{
			UserID:      assignee.ID,
			Username:    assignee.Username,
			DisplayName: assignee.DisplayName,
			Email:       assignee.Email,
			AvatarURL:   assignee.AvatarURL,
		},
	})

	var notificationType string
	if isAdded {
		notificationType = NT_ASSIGNEE_ADDED
	} else {
		notificationType = NT_ASSIGNEE_REMOVED
	}

	for _, assignee := range task.Assignees {
		_, err := s.notificationRepo.Create(
			ctx,
			assignee.AssigneeID,
			notificationType,
			body,
			false,
		)
		if err != nil {
			return fmt.Errorf("notification repository Create: %w", err)
		}
	}

	return nil
}

func (s *NotificationService) JoinRequested(ctx context.Context,
	projectID string,
	requestorID string) (*JoinRequested, error) {

	project, err := s.projectRepo.Get(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("project repository Get: %w", err)
	}

	requestor, err := s.userRepo.Get(ctx, requestorID)
	if err != nil {
		return nil, fmt.Errorf("user repository Get: %w", err)
	}

	data := JoinRequested{
		Project: ProjectBody{
			ID:   projectID,
			Name: project.Name,
		},
		Requestor: core.Avatar{
			UserID:      requestor.ID,
			Username:    requestor.Username,
			DisplayName: requestor.DisplayName,
			Email:       requestor.Email,
			AvatarURL:   requestor.AvatarURL,
		},
	}
	body, _ := json.Marshal(data)

	_, err = s.notificationRepo.Create(
		ctx,
		project.OwnerID,
		NT_JOIN_REQUESTED,
		body,
		false,
	)
	if err != nil {
		return nil, fmt.Errorf("notification repository Create: %w", err)
	}

	return &data, nil
}

func (s *NotificationService) JoinResponded(ctx context.Context,
	projectID string,
	requestorID string,
	status string) error {

	project, err := s.projectRepo.Get(ctx, projectID)
	if err != nil {
		return fmt.Errorf("project repository Get: %w", err)
	}

	responder, err := s.userRepo.Get(ctx, project.OwnerID)
	if err != nil {
		return fmt.Errorf("user repository Get: %w", err)
	}

	body, _ := json.Marshal(JoinResponded{
		Project: ProjectBody{
			ID:   projectID,
			Name: project.Name,
		},
		Responder: core.Avatar{
			UserID:      responder.ID,
			Username:    responder.Username,
			DisplayName: responder.DisplayName,
			Email:       responder.Email,
			AvatarURL:   responder.AvatarURL,
		},
		Status: status,
	})

	_, err = s.notificationRepo.Create(
		ctx,
		requestorID,
		NT_JOIN_RESPONDED,
		body,
		false,
	)
	if err != nil {
		return fmt.Errorf("notification repository Create: %w", err)
	}

	return nil
}

func (s *NotificationService) CommentAdded(ctx context.Context,
	projectID, taskID string,
	commenterID string) error {

	project, err := s.projectRepo.Get(ctx, projectID)
	if err != nil {
		return fmt.Errorf("project repository Get: %w", err)
	}

	task, err := s.taskRepo.Get(ctx, taskID)
	if err != nil {
		return fmt.Errorf("task repository Get: %w", err)
	}

	commenter, err := s.userRepo.Get(ctx, commenterID)
	if err != nil {
		return fmt.Errorf("user repository Get: %w", err)
	}

	body, _ := json.Marshal(CommentAdded{
		Project: ProjectBody{
			ID:   projectID,
			Name: project.Name,
		},
		Task: TaskBody{
			ID:    taskID,
			Title: task.Title,
		},
		Commenter: core.Avatar{
			UserID:      commenter.ID,
			Username:    commenter.Username,
			DisplayName: commenter.DisplayName,
			Email:       commenter.Email,
			AvatarURL:   commenter.AvatarURL,
		},
	})

	for _, assignee := range task.Assignees {
		_, err = s.notificationRepo.Create(
			ctx,
			assignee.AssigneeID,
			NT_COMMENT_ADDED,
			body,
			false,
		)
		if err != nil {
			return fmt.Errorf("notification repository Create: %w", err)
		}
	}

	return nil
}

func (s *NotificationService) List(ctx context.Context,
	userID string) ([]Notification, error) {

	rows, err := s.notificationRepo.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("notification repository List: %w", err)
	}

	notifications := []Notification{}
	var body any
	for _, r := range rows {
		json.Unmarshal(r.Body, &body)
		notifications = append(notifications, Notification{
			ID:        r.ID,
			UserID:    r.UserID,
			Type:      r.Type,
			Body:      body,
			Read:      r.Read,
			CreatedAt: r.CreatedAt,
		})
	}

	return notifications, nil
}

func (s *NotificationService) MarkAsRead(ctx context.Context,
	userID, notificationID string) error {

	err := s.notificationRepo.Update(ctx, notificationID, true)
	if err != nil {
		return fmt.Errorf("notification repository Update: %w", err)
	}

	return nil
}
