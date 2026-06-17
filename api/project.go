package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/core/members"
	"github.com/Arup3201/torb/core/projects"
	"github.com/Arup3201/torb/core/requests"
	"github.com/Arup3201/torb/core/users"
	"github.com/Arup3201/torb/notifications"
	"github.com/Arup3201/torb/realtime"
	"github.com/go-playground/validator/v10"
)

type CreateProjectRequest struct {
	Name        string  `json:"name" validate:"required"`
	Description *string `json:"description"`
	Skills      *string `json:"skills"`
}

type ProjectDetail struct {
	projects.ProjectSummary
	Role        string      `json:"role"`
	MemberCount int64       `json:"members_count"`
	Owner       core.Avatar `json:"owner"`
}

type PublicProjectDetail struct {
	projects.ProjectSummary
	Owner      core.Avatar `json:"owner"`
	JoinStatus string      `json:"join_status"`
}

type ListedProjectSummaries struct {
	Projects []projects.ProjectSummary `json:"projects"`
	Page     int                       `json:"page"`
	Limit    int                       `json:"limit"`
	HasNext  bool                      `json:"has_next"`
}

type ListedProjectPreviews struct {
	Projects []projects.ProjectPreview `json:"projects"`
	Page     int                       `json:"page"`
	Limit    int                       `json:"limit"`
	HasNext  bool                      `json:"has_next"`
}

type ListedJoinRequests struct {
	JoinRequests []requests.JoinRequest `json:"join_requests"`
}

type ListedMembers struct {
	Members []members.Member `json:"members"`
}

type UpdateJoinRequest struct {
	UserID     string `json:"user_id" validate:"required"`
	JoinStatus string `json:"join_status" validate:"required"`
}

type ProjectApi struct {
	notifier            *realtime.WebSocketNotifier
	projectService      *projects.ProjectService
	userService         *users.UserService
	memberService       *members.MemberService
	joinRequestService  *requests.JoinRequestService
	notificationService *notifications.NotificationService
}

func NewProjectApi(
	projectService *projects.ProjectService,
	userService *users.UserService,
	memberService *members.MemberService,
	joinRequestService *requests.JoinRequestService,
	notificationService *notifications.NotificationService,
) *ProjectApi {
	return &ProjectApi{
		notifier:            realtime.GlobalWebSocketNotifier(),
		projectService:      projectService,
		userService:         userService,
		memberService:       memberService,
		joinRequestService:  joinRequestService,
		notificationService: notificationService,
	}
}

func (api *ProjectApi) Create(w http.ResponseWriter, r *http.Request) error {

	var payload CreateProjectRequest

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&payload); err != nil {
		return fmt.Errorf("payload decode: %w", core.ErrInvalidValue)
	}
	if err := validator.New().Struct(payload); err != nil {
		return fmt.Errorf("payload validation: %w", core.ErrInvalidValue)
	}

	userID, err := GetUserID(r)
	if err != nil {
		return fmt.Errorf("get user Id: %w", err)
	}

	projectID, err := api.projectService.Create(r.Context(),
		payload.Name, payload.Description, payload.Skills, userID)
	if err != nil {
		return fmt.Errorf("store create project: %w", err)
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(HTTPSuccessResponse[string]{
		Status: RESPONSE_SUCCESS_STATUS,
		Data:   &projectID,
	})
	return nil
}

func (api *ProjectApi) Get(w http.ResponseWriter, r *http.Request) error {

	projectID := r.PathValue("id")
	if projectID == "" {
		return core.ErrInvalidValue
	}

	projectSummary, err := api.projectService.Get(r.Context(), projectID)
	if err != nil {
		return fmt.Errorf("database get project details: %w", err)
	}

	owner, err := api.userService.Get(r.Context(), projectSummary.OwnerID)
	if err != nil {
		return fmt.Errorf("user service get: %w", err)
	}

	userID, err := GetUserID(r)
	if err != nil {
		return fmt.Errorf("get context user: %w", err)
	}

	role, err := api.memberService.GetRole(r.Context(), projectID, userID)
	if !errors.Is(err, core.ErrNotFound) && err != nil {
		return fmt.Errorf("member service get role: %w", err)
	}

	if role == core.ROLE_MEMBER || role == core.ROLE_OWNER {
		memberCount, err := api.memberService.Count(r.Context(), projectID, userID)
		if err != nil {
			return fmt.Errorf("member service count: %w", err)
		}

		json.NewEncoder(w).Encode(HTTPSuccessResponse[ProjectDetail]{
			Status: RESPONSE_SUCCESS_STATUS,
			Data: &ProjectDetail{
				ProjectSummary: *projectSummary,
				MemberCount:    memberCount,
				Role:           role,
				Owner: core.Avatar{
					UserID:      owner.ID,
					Username:    owner.Username,
					Email:       owner.Email,
					DisplayName: owner.DisplayName,
					AvatarURL:   owner.AvatarURL,
				},
			},
		})
	} else {
		joinStatus, err := api.joinRequestService.GetStatus(r.Context(),
			projectID, userID)
		if err != nil {
			return fmt.Errorf("join request service get status: %w", err)
		}

		json.NewEncoder(w).Encode(HTTPSuccessResponse[PublicProjectDetail]{
			Status: RESPONSE_SUCCESS_STATUS,
			Data: &PublicProjectDetail{
				ProjectSummary: *projectSummary,
				Owner: core.Avatar{
					UserID:      owner.ID,
					Username:    owner.Username,
					Email:       owner.Email,
					DisplayName: owner.DisplayName,
					AvatarURL:   owner.AvatarURL,
				},
				JoinStatus: joinStatus,
			},
		})
	}

	return nil
}

func (api *ProjectApi) ListMyProjects(w http.ResponseWriter, r *http.Request) error {
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

	userID, err := GetUserID(r)
	if err != nil {
		return fmt.Errorf("get userID: %w", err)
	}

	summaries, err := api.projectService.MyProjects(r.Context(), userID)
	if err != nil {
		return fmt.Errorf("project service my projects: %w", err)
	}

	json.NewEncoder(w).Encode(HTTPSuccessResponse[ListedProjectSummaries]{
		Status: RESPONSE_SUCCESS_STATUS,
		Data: &ListedProjectSummaries{
			Projects: summaries,
			Page:     page,
			Limit:    limit,
		},
	})

	return nil
}

func (api *ProjectApi) ListPublic(w http.ResponseWriter, r *http.Request) error {
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

	userID, err := GetUserID(r)
	if err != nil {
		return fmt.Errorf("get userID: %w", err)
	}

	projects, err := api.projectService.ListPublic(r.Context(), userID)
	if err != nil {
		return fmt.Errorf("project service list public: %w", err)
	}

	json.NewEncoder(w).Encode(HTTPSuccessResponse[ListedProjectPreviews]{
		Status: RESPONSE_SUCCESS_STATUS,
		Data: &ListedProjectPreviews{
			Projects: projects,
			Page:     page,
			Limit:    limit,
		},
	})
	return nil
}

func (api *ProjectApi) ListRecentlyCreated(w http.ResponseWriter, r *http.Request) error {

	userID, err := GetUserID(r)
	if err != nil {
		return fmt.Errorf("get userID: %w", err)
	}

	projects, err := api.projectService.RecentlyCreated(r.Context(), userID)
	if err != nil {
		return fmt.Errorf("project service recently created: %w", err)
	}

	json.NewEncoder(w).Encode(HTTPSuccessResponse[ListedProjectSummaries]{
		Status: RESPONSE_SUCCESS_STATUS,
		Data: &ListedProjectSummaries{
			Projects: projects,
		},
	})

	return nil
}

func (api *ProjectApi) ListRecentlyJoined(w http.ResponseWriter, r *http.Request) error {
	userID, err := GetUserID(r)
	if err != nil {
		return fmt.Errorf("get userID: %w", err)
	}

	projects, err := api.projectService.RecentlyJoined(r.Context(), userID)
	if err != nil {
		return fmt.Errorf("project service recently joined: %w", err)
	}

	json.NewEncoder(w).Encode(HTTPSuccessResponse[ListedProjectSummaries]{
		Status: RESPONSE_SUCCESS_STATUS,
		Data: &ListedProjectSummaries{
			Projects: projects,
		},
	})

	return nil
}

func (api *ProjectApi) ListMembers(w http.ResponseWriter, r *http.Request) error {

	projectID := r.PathValue("id")
	if projectID == "" {
		return core.ErrInvalidValue
	}

	userID, err := GetUserID(r)
	if err != nil {
		return fmt.Errorf("get userID: %w", err)
	}

	members, err := api.memberService.AllMembers(r.Context(),
		projectID,
		userID)
	if err != nil {
		return fmt.Errorf("member service all members: %w", err)
	}

	json.NewEncoder(w).Encode(HTTPSuccessResponse[ListedMembers]{
		Status: RESPONSE_SUCCESS_STATUS,
		Data: &ListedMembers{
			Members: members,
		},
	})

	return nil
}

func (api *ProjectApi) AddJoinRequest(w http.ResponseWriter, r *http.Request) error {

	projectID := r.PathValue("id")
	if projectID == "" {
		return core.ErrInvalidValue
	}

	userID, err := GetUserID(r)
	if err != nil {
		return fmt.Errorf("get projects userID: %w", err)
	}

	err = api.joinRequestService.Create(r.Context(), projectID, userID)
	if err != nil {
		return fmt.Errorf("join request service create: %w", err)
	}

	jsonData, err := api.notificationService.JoinRequested(
		r.Context(),
		projectID,
		userID,
	)
	if err != nil {
		log.Printf("[ERROR] notification service JoinRequested: %s", err)
	}

	api.notifier.Notify(userID, jsonData)
	json.NewEncoder(w).Encode(HTTPSuccessResponse[any]{
		Status:  RESPONSE_SUCCESS_STATUS,
		Message: "Join request created",
	})

	return nil
}

func (api *ProjectApi) ListJoinRequests(w http.ResponseWriter, r *http.Request) error {

	projectID := r.PathValue("id")
	if projectID == "" {
		return fmt.Errorf("project ID is missing: %w", core.ErrInvalidValue)
	}

	userID, err := GetUserID(r)
	if err != nil {
		return fmt.Errorf("get userID: %w", err)
	}

	joinRequests, err := api.joinRequestService.List(r.Context(),
		projectID,
		userID)
	if err != nil {
		return fmt.Errorf("join request service list: %w", err)
	}

	json.NewEncoder(w).Encode(HTTPSuccessResponse[ListedJoinRequests]{
		Status: RESPONSE_SUCCESS_STATUS,
		Data: &ListedJoinRequests{
			JoinRequests: joinRequests,
		},
	})

	return nil
}

func (api *ProjectApi) RespondToJoinRequest(w http.ResponseWriter, r *http.Request) error {

	projectID := r.PathValue("id")
	if projectID == "" {
		return core.ErrInvalidValue
	}

	var payload UpdateJoinRequest

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&payload); err != nil {
		return core.ErrInvalidValue
	}
	if err := validator.New().Struct(payload); err != nil {
		return core.ErrInvalidValue
	}

	if payload.UserID == "" {
		return core.ErrInvalidValue
	}
	if payload.JoinStatus == "" {
		return core.ErrInvalidValue
	}

	userID, err := GetUserID(r)
	if err != nil {
		return fmt.Errorf("get projects userID: %w", err)
	}

	err = api.joinRequestService.Respond(
		r.Context(),
		projectID,
		userID,
		payload.UserID,
		payload.JoinStatus,
	)
	if err != nil {
		return fmt.Errorf("join request service respond: %w", err)
	}

	data, err := api.notificationService.JoinResponded(
		r.Context(),
		projectID,
		payload.UserID,
		payload.JoinStatus,
	)
	if err != nil {
		log.Printf("[ERROR] notification service JoinResponded: %s", err)
	}

	api.notifier.Notify(payload.UserID, data)

	json.NewEncoder(w).Encode(HTTPSuccessResponse[any]{
		Status:  RESPONSE_SUCCESS_STATUS,
		Message: "Join request status updated",
	})

	return nil
}
