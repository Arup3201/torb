package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/core/assignees"
	"github.com/Arup3201/torb/core/comments"
	"github.com/Arup3201/torb/core/tasks"
	"github.com/Arup3201/torb/notifications"
	"github.com/go-playground/validator/v10"
)

type CreateTaskRequest struct {
	Title       string   `json:"title" validate:"required"`
	Description string   `json:"description" validate:"required"`
	Assignees   []string `json:"assignees" validate:"required"`
	Status      string   `json:"status" validate:"required"`
}

type CreatedTaskResponse struct {
	TaskID   string   `json:"task_id"`
	Warnings []string `json:"warnings"`
}

type UpdateTaskRequest struct {
	Title             *string  `json:"title"`
	Description       *string  `json:"description"`
	Status            *string  `json:"status"`
	AssigneesToAdd    []string `json:"assignees_to_add"`
	AssigneesToRemove []string `json:"assignees_to_remove"`
}

type UpdateTaskResponse struct {
	Warnings []string `json:"warnings"`
}

type ListedTasks struct {
	Tasks   []tasks.ProjectTaskItem `json:"tasks"`
	Page    int                     `json:"page"`
	Limit   int                     `json:"limit"`
	HasNext bool                    `json:"has_next"`
}

type ListedDashboardTasks struct {
	Tasks   []tasks.DashboardTaskItem `json:"tasks"`
	Page    int                       `json:"page"`
	Limit   int                       `json:"limit"`
	HasNext bool                      `json:"has_next"`
}

type AddCommentRequest struct {
	UserId  string `json:"user_id"`
	Comment string `json:"comment"`
}

type ListedComments struct {
	Comments []comments.Comment `json:"comments"`
	Page     int                `json:"page"`
	Limit    int                `json:"limit"`
	HasNext  bool               `json:"has_next"`
}

type TaskApi struct {
	taskService         *tasks.TaskService
	assigneeService     *assignees.AssigneeService
	commentService      *comments.CommentService
	notificationService *notifications.NotificationService
}

func NewTaskApi(
	taskService *tasks.TaskService,
	assigneeService *assignees.AssigneeService,
	commentService *comments.CommentService,
	notificationService *notifications.NotificationService,
) *TaskApi {
	return &TaskApi{
		taskService:         taskService,
		assigneeService:     assigneeService,
		commentService:      commentService,
		notificationService: notificationService,
	}
}

func (api *TaskApi) Create(w http.ResponseWriter, r *http.Request) error {

	projectID := r.PathValue("project_id")
	if projectID == "" {
		return core.ErrInvalidValue
	}

	var payload CreateTaskRequest

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&payload); err != nil {
		return core.ErrInvalidValue
	}
	if err := validator.New().Struct(payload); err != nil {
		return core.ErrInvalidValue
	}

	userID, err := GetUserID(r)
	if err != nil {
		return fmt.Errorf("get userID: %w", err)
	}

	taskID, err := api.taskService.Create(
		r.Context(),
		projectID,
		userID,
		payload.Title,
		payload.Description,
		payload.Status)
	if err != nil {
		return fmt.Errorf("service create task: %w", err)
	}

	err = api.notificationService.TaskAdded(
		r.Context(),
		projectID,
		taskID,
	)
	if err != nil {
		log.Printf("[ERROR] notification service TaskAdded: %s", err)
	}

	warnings := []string{}
	for _, assignee := range payload.Assignees {
		err = api.assigneeService.AddAssignee(r.Context(),
			projectID,
			taskID,
			userID,
			assignee)
		if err != nil {
			warnings = append(warnings, err.Error())
		} else {
			err = api.notificationService.AssigneeUpdated(
				r.Context(),
				projectID,
				taskID,
				assignee,
				true,
			)
			if err != nil {
				log.Printf("[ERROR] notification service AssigneeUpdated: %s", err)
			}
		}
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(HTTPSuccessResponse[CreatedTaskResponse]{
		Status: RESPONSE_SUCCESS_STATUS,
		Data: &CreatedTaskResponse{
			TaskID:   taskID,
			Warnings: warnings,
		},
	})

	return nil
}

func (api *TaskApi) Get(w http.ResponseWriter, r *http.Request) error {

	projectID := r.PathValue("project_id")
	if projectID == "" {
		return core.ErrInvalidValue
	}

	taskID := r.PathValue("task_id")
	if taskID == "" {
		return core.ErrInvalidValue
	}

	userID, err := GetUserID(r)
	if err != nil {
		return fmt.Errorf("get userID: %w", err)
	}

	task, err := api.taskService.Get(
		r.Context(),
		projectID,
		taskID,
		userID)
	if err != nil {
		return fmt.Errorf("service get task: %w", err)
	}

	json.NewEncoder(w).Encode(HTTPSuccessResponse[tasks.ProjectTaskItem]{
		Status: RESPONSE_SUCCESS_STATUS,
		Data:   task,
	})

	return nil
}

func (api *TaskApi) Update(w http.ResponseWriter, r *http.Request) error {

	projectID := r.PathValue("project_id")
	if projectID == "" {
		return core.ErrInvalidValue
	}

	taskID := r.PathValue("task_id")
	if taskID == "" {
		return core.ErrInvalidValue
	}

	var payload UpdateTaskRequest

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&payload); err != nil {
		return core.ErrInvalidValue
	}
	if err := validator.New().Struct(payload); err != nil {
		return core.ErrInvalidValue
	}

	userID, err := GetUserID(r)
	if err != nil {
		return fmt.Errorf("get userID: %w", err)
	}

	if payload.Title != nil || payload.Description != nil || payload.Status != nil {
		err = api.taskService.Update(r.Context(),
			projectID,
			taskID,
			userID,
			payload.Title,
			payload.Description,
			payload.Status,
		)

		if err != nil {
			return fmt.Errorf("service task update: %w", err)
		} else {
			err = api.notificationService.TaskUpdated(
				r.Context(),
				projectID,
				taskID,
				payload.Title,
				payload.Description,
				payload.Status,
				userID,
			)
			if err != nil {
				log.Printf("[ERROR] notification service TaskUpdated: %s", err)
			}
		}
	}

	warnings := []string{}
	for _, assignee := range payload.AssigneesToAdd {
		err = api.assigneeService.AddAssignee(r.Context(),
			projectID, taskID, userID, assignee)
		if err != nil {
			warnings = append(warnings, err.Error())
		} else {
			err = api.notificationService.AssigneeUpdated(
				r.Context(),
				projectID,
				taskID,
				assignee,
				true,
			)
			if err != nil {
				log.Printf("[ERROR] notification service AssigneeUpdated: %s", err)
			}
		}
	}
	for _, assignee := range payload.AssigneesToRemove {
		err = api.assigneeService.RemoveAssignee(r.Context(),
			projectID, taskID, userID, assignee)
		if err != nil {
			warnings = append(warnings, err.Error())
		} else {
			err = api.notificationService.AssigneeUpdated(
				r.Context(),
				projectID,
				taskID,
				assignee,
				false,
			)
			if err != nil {
				log.Printf("[ERROR] notification service AssigneeUpdated: %s", err)
			}
		}
	}

	json.NewEncoder(w).Encode(HTTPSuccessResponse[UpdateTaskResponse]{
		Status: RESPONSE_SUCCESS_STATUS,
		Data: &UpdateTaskResponse{
			Warnings: warnings,
		},
		Message: "Task updated successfully",
	})

	return nil
}

func (api *TaskApi) List(w http.ResponseWriter, r *http.Request) error {

	queryPage := r.URL.Query().Get("page")
	queryLimit := r.URL.Query().Get("limit")

	var page, limit int
	if queryPage != "" {
		var err error
		page, err = strconv.Atoi(queryPage)
		if err != nil {
			return core.ErrInvalidValue
		}
	} else {
		page = 1
	}
	if queryLimit != "" {
		var err error
		limit, err = strconv.Atoi(queryLimit)
		if err != nil {
			return core.ErrInvalidValue
		}
	} else {
		limit = 10
	}

	projectID := r.PathValue("project_id")
	if projectID == "" {
		return core.ErrInvalidValue
	}

	userID, err := GetUserID(r)
	if err != nil {
		return fmt.Errorf("get context user: %w", err)
	}

	tasks, err := api.taskService.List(r.Context(), projectID, userID)
	if err != nil {
		return fmt.Errorf("service list tasks: %w", err)
	}

	// TODO: count total tasks and check has next
	hasNext := false

	json.NewEncoder(w).Encode(HTTPSuccessResponse[ListedTasks]{
		Status: RESPONSE_SUCCESS_STATUS,
		Data: &ListedTasks{
			Tasks:   tasks,
			Page:    page,
			Limit:   limit,
			HasNext: hasNext,
		},
	})

	return nil
}

func (api *TaskApi) AddComment(w http.ResponseWriter, r *http.Request) error {

	projectID := r.PathValue("project_id")
	if projectID == "" {
		return core.ErrInvalidValue
	}

	taskID := r.PathValue("task_id")
	if taskID == "" {
		return core.ErrInvalidValue
	}

	var payload AddCommentRequest

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&payload); err != nil {
		return core.ErrInvalidValue
	}
	if err := validator.New().Struct(payload); err != nil {
		return core.ErrInvalidValue
	}

	userID, err := GetUserID(r)
	if err != nil {
		return fmt.Errorf("get userID: %w", err)
	}

	commentId, err := api.commentService.Create(
		r.Context(),
		projectID,
		taskID,
		userID,
		payload.Comment,
	)
	if err != nil {
		return fmt.Errorf("comment service create: %w", err)
	}

	err = api.notificationService.CommentAdded(
		r.Context(),
		projectID,
		taskID,
		userID,
	)
	if err != nil {
		log.Printf("[ERROR] notification service CommentAdded: %s", err)
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(HTTPSuccessResponse[string]{
		Status: RESPONSE_SUCCESS_STATUS,
		Data:   &commentId,
	})

	return nil
}

func (api *TaskApi) ListComments(w http.ResponseWriter, r *http.Request) error {

	projectID := r.PathValue("project_id")
	if projectID == "" {
		return core.ErrInvalidValue
	}

	taskID := r.PathValue("task_id")
	if taskID == "" {
		return core.ErrInvalidValue
	}

	userID, err := GetUserID(r)
	if err != nil {
		return fmt.Errorf("get userID: %w", err)
	}

	comments, err := api.commentService.List(
		r.Context(),
		projectID,
		taskID,
		userID,
	)
	if err != nil {
		return fmt.Errorf("comment service list: %w", err)
	}

	json.NewEncoder(w).Encode(HTTPSuccessResponse[ListedComments]{
		Status: RESPONSE_SUCCESS_STATUS,
		Data: &ListedComments{
			Comments: comments,
			Page:     1,
			Limit:    len(comments),
			HasNext:  false,
		},
	})

	return nil
}

func (api *TaskApi) ListAssignedTasks(w http.ResponseWriter, r *http.Request) error {

	userID, err := GetUserID(r)
	if err != nil {
		return fmt.Errorf("get userID: %w", err)
	}

	tasks, err := api.taskService.RecentlyAssigned(r.Context(),
		userID)
	if err != nil {
		return fmt.Errorf("task service assigned tasks: %w", err)
	}

	json.NewEncoder(w).Encode(HTTPSuccessResponse[ListedDashboardTasks]{
		Status: RESPONSE_SUCCESS_STATUS,
		Data: &ListedDashboardTasks{
			Tasks:   tasks,
			Page:    1,
			Limit:   len(tasks),
			HasNext: false,
		},
	})

	return nil
}

func (api *TaskApi) ListUnassignedTasks(w http.ResponseWriter, r *http.Request) error {

	userID, err := GetUserID(r)
	if err != nil {
		return fmt.Errorf("get userID: %w", err)
	}

	tasks, err := api.taskService.RecentlyUnassigned(r.Context(),
		userID)
	if err != nil {
		return fmt.Errorf("task service unassigned tasks: %w", err)
	}

	json.NewEncoder(w).Encode(HTTPSuccessResponse[ListedDashboardTasks]{
		Status: RESPONSE_SUCCESS_STATUS,
		Data: &ListedDashboardTasks{
			Tasks:   tasks,
			Page:    1,
			Limit:   len(tasks),
			HasNext: false,
		},
	})

	return nil
}
