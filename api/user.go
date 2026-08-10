package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Arup3201/torb/auth/manual"
	"github.com/Arup3201/torb/auth/openid"
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
	Timezone       string    `json:"timezone"`
	LoginMethod    string    `json:"login_method"`
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
	Timezone    *string `json:"timezone"`
}

type CreatePasswordAccountRequest struct {
	Password string `json:"password"`
}

type UserApi struct {
	userService     *users.UserService
	googleService   *openid.GoogleService
	registerService *manual.RegisterService
}

func NewUserApi(
	userService *users.UserService,
	googleService *openid.GoogleService,
	registerService *manual.RegisterService,
) *UserApi {
	return &UserApi{
		userService:     userService,
		googleService:   googleService,
		registerService: registerService,
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

	hasManualAccount, err := api.registerService.HasAccount(r.Context(), userID)
	if err != nil {
		return fmt.Errorf("register service HasAccount: %w", err)
	}

	hasGoogleAccount, err := api.googleService.HasAccount(r.Context(), userID)
	if err != nil {
		return fmt.Errorf("google service HasAccount: %w", err)
	}

	var loginMethod string
	if hasManualAccount && hasGoogleAccount {
		loginMethod = "both"
	} else if hasGoogleAccount {
		loginMethod = "google"
	} else if hasManualAccount {
		loginMethod = "password"
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
			Timezone:       userData.Timezone,
			LoginMethod:    loginMethod, // password / google / both
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

	json.NewEncoder(w).Encode(HTTPSuccessResponse[any]{
		Status:  RESPONSE_SUCCESS_STATUS,
		Message: "User data updated",
	})

	return nil
}

func (api *UserApi) CreatePasswordAccount(w http.ResponseWriter, r *http.Request) error {

	var payload CreatePasswordAccountRequest

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

	err = api.registerService.AddPassword(r.Context(), userID, payload.Password)
	if err != nil {
		return fmt.Errorf("add password service: %w", err)
	}

	json.NewEncoder(w).Encode(HTTPSuccessResponse[any]{
		Status:  RESPONSE_SUCCESS_STATUS,
		Message: "User can now login using this password",
	})

	return nil
}
