package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Arup3201/torb/core/users"
)

type UserProfileSummary struct {
	ID             string    `json:"id"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	DisplayName    *string   `json:"display_name"`
	AvatarURL      *string   `json:"avatar_url"`
	Skills         string    `json:"skills"`
	Projects       int64     `json:"projects"`
	Tasks          int64     `json:"tasks"`
	CompletedTasks int64     `json:"completed_tasks"`
	UserSince      time.Time `json:"user_since"`
	LastLoginTime  time.Time `json:"last_login_time"`
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
			Projects:       userSummary.Projects,
			Tasks:          userSummary.Tasks,
			CompletedTasks: userSummary.CompletedTasks,
			UserSince:      userData.CreatedAt,
			LastLoginTime:  userData.LastLoginTime,
		},
	})

	return nil
}
