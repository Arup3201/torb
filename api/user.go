package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/core/users"
	"github.com/go-playground/validator/v10"
)

type UserProfileSummary struct {
	ID             string    `json:"id"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	DisplayName    *string   `json:"display_name"`
	AvatarURL      *string   `json:"avatar_url"`
	Skills         string    `json:"skills"`
	Timestamp      string    `json:"timestamp"`
	Projects       int64     `json:"projects"`
	Tasks          int64     `json:"tasks"`
	CompletedTasks int64     `json:"completed_tasks"`
	UserSince      time.Time `json:"user_since"`
	LastLoginTime  time.Time `json:"last_login_time"`
}

type UpdateUserRequest struct {
	Username    *string `json:"username"`
	DisplayName *string `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
	Skills      *string `json:"skills"`
	Timestamp   *string `json:"timestamp"`
}

type UserApi struct {
	userService *users.UserService
}

func NewUserApi(
	userService *users.UserService,
) *UserApi {
	return &UserApi{
		userService: userService,
	}
}

func (api *UserApi) GetProfileSummary(w http.ResponseWriter, r *http.Request) error {

	userID, err := GetUserID(r)
	if err != nil {
		return fmt.Errorf("get user Id: %w", err)
	}

	userData, err := api.userService.Get(r.Context(), userID)
	if err != nil {
		return fmt.Errorf("user service Get: %w", err)
	}

	userSummary, err := api.userService.ProjectAndTaskCount(r.Context(), userID)
	if err != nil {
		return fmt.Errorf("user service ProjectAndTaskCount: %w", err)
	}

	json.NewEncoder(w).Encode(HTTPSuccessResponse[UserProfileSummary]{
		Status: RESPONSE_SUCCESS_STATUS,
		Data: &UserProfileSummary{
			ID:             userData.ID,
			Username:       userData.Username,
			Email:          userData.Email,
			DisplayName:    userData.DisplayName,
			AvatarURL:      userData.AvatarURL,
			Skills:         userData.Skills,
			Timestamp:      userData.Timestamp,
			Projects:       userSummary.Projects,
			Tasks:          userSummary.Tasks,
			CompletedTasks: userSummary.CompletedTasks,
			UserSince:      userData.CreatedAt,
			LastLoginTime:  userData.LastLoginTime,
		},
	})

	return nil
}

func (api *UserApi) UpdateProfile(w http.ResponseWriter, r *http.Request) error {

	var payload UpdateUserRequest

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

	err = api.userService.Update(r.Context(), userID, users.UpdateUserBody(payload))
	if err != nil {
		return fmt.Errorf("user service Update: %w", err)
	}

	return nil
}
