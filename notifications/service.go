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
	projectID, taskID string) ([]byte, []string, error) {

	project, err := s.projectRepo.Get(ctx, projectID)
	if err != nil {
		return nil, nil, fmt.Errorf("project repository Get: %w", err)
	}

	task, err := s.taskRepo.Get(ctx, taskID)
	if err != nil {
		return nil, nil, fmt.Errorf("task repository Get: %w", err)
	}

	members, err := s.membershipRepo.List(ctx, projectID)
	if err != nil {
		return nil, nil, fmt.Errorf("membership repository Get: %w", err)
	}

	taskAddedData := TaskAdded{
		Project: ProjectBody{
			ID:   projectID,
			Name: project.Name,
		},
		Task: TaskBody{
			ID:    taskID,
			Title: task.Title,
		},
	}
	body, _ := json.Marshal(taskAddedData)
	for _, m := range members {
		if m.Role.String == core.ROLE_OWNER {
			continue
		}

		_, err := s.notificationRepo.Create(ctx, m.UserID, NT_TASK_ADDED, body, false)
		if err != nil {
			return nil, nil, fmt.Errorf("notification repository Create: %w", err)
		}
	}

	jsonData, _ := json.Marshal(struct {
		Type string    `json:"type"`
		Data TaskAdded `json:"data"`
	}{
		Type: NT_TASK_ADDED,
		Data: taskAddedData,
	})

	notificaionReceivers := []string{}
	for _, m := range members {
		notificaionReceivers = append(notificaionReceivers, m.UserID)
	}

	return jsonData, notificaionReceivers, nil
}

func (s *NotificationService) TaskUpdated(ctx context.Context,
	projectID, taskID string,
	title, description, status *string,
	updaterID string) ([]byte, []string, error) {

	project, err := s.projectRepo.Get(ctx, projectID)
	if err != nil {
		return nil, nil, fmt.Errorf("project repository Get: %w", err)
	}

	task, err := s.taskRepo.Get(ctx, taskID)
	if err != nil {
		return nil, nil, fmt.Errorf("task repository Get: %w", err)
	}

	updater, err := s.userRepo.Get(ctx, updaterID)
	if err != nil {
		return nil, nil, fmt.Errorf("user repository Get: %w", err)
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

	taskUpdatedData := TaskUpdated{
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
	}
	body, _ := json.Marshal(taskUpdatedData)

	if project.OwnerID != updaterID {
		_, err := s.notificationRepo.Create(
			ctx,
			project.OwnerID,
			NT_TASK_UPDATED,
			body,
			false,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("notification repository Create: %w", err)
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
			return nil, nil, fmt.Errorf("notification repository Create: %w", err)
		}
	}

	jsonData, _ := json.Marshal(struct {
		Type string      `json:"type"`
		Data TaskUpdated `json:"data"`
	}{
		Type: NT_TASK_UPDATED,
		Data: taskUpdatedData,
	})

	receivers := []string{}
	for _, m := range task.Assignees {
		receivers = append(receivers, m.AssigneeID)
	}

	return jsonData, receivers, nil
}

func (s *NotificationService) AssigneeUpdated(ctx context.Context,
	projectID, taskID string,
	assigneeID string,
	isAdded bool) ([]byte, error) {

	project, err := s.projectRepo.Get(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("project repository Get: %w", err)
	}

	task, err := s.taskRepo.Get(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("task repository Get: %w", err)
	}

	assignee, err := s.userRepo.Get(ctx, assigneeID)
	if err != nil {
		return nil, fmt.Errorf("user repository Get: %w", err)
	}

	taskAssignedData := AssigneeUpdated{
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
	}
	body, _ := json.Marshal(taskAssignedData)

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
			return nil, fmt.Errorf("notification repository Create: %w", err)
		}
	}

	jsonData, _ := json.Marshal(struct {
		Type string          `json:"type"`
		Data AssigneeUpdated `json:"data"`
	}{
		Type: notificationType,
		Data: taskAssignedData,
	})

	return jsonData, nil
}

func (s *NotificationService) JoinRequested(ctx context.Context,
	projectID string,
	requestorID string) ([]byte, error) {

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

	jsonData, err := json.Marshal(struct {
		Type string        `json:"type"`
		Data JoinRequested `json:"data"`
	}{
		Type: NT_JOIN_REQUESTED,
		Data: data,
	})

	return jsonData, nil
}

func (s *NotificationService) JoinResponded(ctx context.Context,
	projectID string,
	requestorID string,
	status string) ([]byte, error) {

	project, err := s.projectRepo.Get(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("project repository Get: %w", err)
	}

	responder, err := s.userRepo.Get(ctx, project.OwnerID)
	if err != nil {
		return nil, fmt.Errorf("user repository Get: %w", err)
	}

	joinRespondedData := JoinResponded{
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
	}
	body, _ := json.Marshal(joinRespondedData)

	_, err = s.notificationRepo.Create(
		ctx,
		requestorID,
		NT_JOIN_RESPONDED,
		body,
		false,
	)
	if err != nil {
		return nil, fmt.Errorf("notification repository Create: %w", err)
	}

	jsonData, _ := json.Marshal(struct {
		Type string        `json:"type"`
		Data JoinResponded `json:"data"`
	}{
		Type: NT_JOIN_RESPONDED,
		Data: joinRespondedData,
	})

	return jsonData, nil
}

func (s *NotificationService) CommentAdded(ctx context.Context,
	projectID, taskID string,
	commenterID string) ([]byte, []string, error) {

	project, err := s.projectRepo.Get(ctx, projectID)
	if err != nil {
		return nil, nil, fmt.Errorf("project repository Get: %w", err)
	}

	task, err := s.taskRepo.Get(ctx, taskID)
	if err != nil {
		return nil, nil, fmt.Errorf("task repository Get: %w", err)
	}

	commenter, err := s.userRepo.Get(ctx, commenterID)
	if err != nil {
		return nil, nil, fmt.Errorf("user repository Get: %w", err)
	}

	commentAddedData := CommentAdded{
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
	}
	body, _ := json.Marshal(commentAddedData)

	for _, assignee := range task.Assignees {
		_, err = s.notificationRepo.Create(
			ctx,
			assignee.AssigneeID,
			NT_COMMENT_ADDED,
			body,
			false,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("notification repository Create: %w", err)
		}
	}

	jsonData, _ := json.Marshal(struct {
		Type string       `json:"type"`
		Data CommentAdded `json:"data"`
	}{
		Type: NT_COMMENT_ADDED,
		Data: commentAddedData,
	})

	receivers := []string{}
	for _, m := range task.Assignees {
		receivers = append(receivers, m.AssigneeID)
	}

	return jsonData, receivers, nil
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
