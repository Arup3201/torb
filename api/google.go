package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/Arup3201/torb/auth"
	"github.com/Arup3201/torb/auth/openid"
	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/core/users"
	"github.com/go-playground/validator/v10"
)

type GoogleLoginRequest struct {
	Token  string `json:"token" validate:"required"`
	UserID string `json:"user_id" validate:"required"`
}

type GoogleApi struct {
	googleService                     *openid.GoogleService
	tokenService                      *auth.TokenService
	userService                       *users.UserService
	frontendHomeURL, frontendLoginURL string
}

func NewGoogleApi(
	googleService *openid.GoogleService,
	tokenService *auth.TokenService,
	userService *users.UserService,
	homeURL, loginURL string,
) *GoogleApi {
	return &GoogleApi{
		googleService:    googleService,
		tokenService:     tokenService,
		userService:      userService,
		frontendHomeURL:  homeURL,
		frontendLoginURL: loginURL,
	}
}

func (api *GoogleApi) Redirect(w http.ResponseWriter, r *http.Request) error {

	url, err := api.googleService.GetAuthCodeURL(r.Context())
	if err != nil {
		return fmt.Errorf("google service GetAuthCodeURL: %w", err)
	}

	http.Redirect(w, r, url, http.StatusSeeOther)

	return nil
}

func (api *GoogleApi) Callback(w http.ResponseWriter, r *http.Request) error {

	errParam := r.URL.Query().Get("error")
	if errParam != "" {
		http.Redirect(w, r, api.frontendLoginURL+"?error="+errParam, http.StatusSeeOther)
		return nil
	}

	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")

	if state == "" {
		http.Redirect(w, r, api.frontendLoginURL+"?error=missing_state", http.StatusSeeOther)
		return nil
	}

	if code == "" {
		http.Redirect(w, r, api.frontendLoginURL+"?error=missing_auth_code", http.StatusSeeOther)
		return nil
	}

	userID, token, err := api.googleService.Callback(
		r.Context(),
		state,
		code,
	)
	if errors.Is(err, core.ErrInvalidValue) {
		log.Printf("[ERROR] google service Callback: %s\n", err)
		http.Redirect(w, r, api.frontendLoginURL+"?error=invalid_state_or_code", http.StatusSeeOther)
		return nil
	} else if errors.Is(err, core.ErrDuplicate) {
		log.Printf("[ERROR] google service Callback: %s\n", err)
		http.Redirect(w, r, api.frontendLoginURL+"?error=account_exist", http.StatusSeeOther)
		return nil
	}
	if err != nil {
		log.Printf("[ERROR] google service Callback: %s\n", err)
		http.Redirect(w, r, api.frontendLoginURL+"?error=server_error", http.StatusSeeOther)
		return nil
	}

	http.Redirect(w, r,
		api.frontendLoginURL+"?user_id="+userID+"&token="+token,
		http.StatusSeeOther)

	return nil
}

func (api *GoogleApi) Login(w http.ResponseWriter, r *http.Request) error {

	var payload GoogleLoginRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&payload); err != nil {
		return fmt.Errorf("decode request body: %w", core.ErrInvalidValue)
	}
	if err := validator.New().Struct(payload); err != nil {
		return fmt.Errorf("validate request body: %w", core.ErrInvalidValue)
	}

	err := api.googleService.ValidToken(
		r.Context(),
		payload.Token,
	)
	if err != nil {
		return err
	}

	refreshToken, err := api.tokenService.CreateRefreshToken(
		r.Context(),
		payload.UserID,
	)
	if err != nil {
		return fmt.Errorf("token service create refresh token: %w", err)
	}

	accessToken, err := api.tokenService.CreateAccessToken(
		r.Context(),
		payload.UserID,
	)
	if err != nil {
		return fmt.Errorf("token service create access token: %w", err)
	}

	user, err := api.userService.Get(
		r.Context(),
		payload.UserID)
	if err != nil {
		return fmt.Errorf("user service get: %w", err)
	}

	cookie := &http.Cookie{
		Name:     REFRESH_TOKEN_COOKIE_NAME,
		Value:    refreshToken.Value,
		Path:     "/", // TODO: auth path only
		Expires:  refreshToken.ExpiresAt,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)

	json.NewEncoder(w).Encode(HTTPSuccessResponse[LoginResponse]{
		Status: RESPONSE_SUCCESS_STATUS,
		Data: &LoginResponse{
			AccessToken: accessToken.Value,
			ExpiresAt:   accessToken.ExpiresAt,
			User: core.Avatar{
				UserID:      user.ID,
				Username:    user.Username,
				Email:       user.Email,
				DisplayName: user.DisplayName,
				AvatarURL:   user.AvatarURL,
			},
		},
	})

	return nil
}

func (api *GoogleApi) Redirect2(w http.ResponseWriter, r *http.Request) error {

	url, err := api.googleService.GetConnectURL(r.Context())
	if err != nil {
		return fmt.Errorf("google service GetConnectURL: %w", err)
	}

	http.Redirect(w, r, url, http.StatusSeeOther)

	return nil
}

func (api *GoogleApi) ConnectGoogleAccountCallback(w http.ResponseWriter, r *http.Request) error {

	redirectURL := api.frontendHomeURL + "profile"

	errParam := r.URL.Query().Get("error")
	if errParam != "" {
		http.Redirect(w, r, redirectURL+"?error="+errParam, http.StatusSeeOther)
		return nil
	}

	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")

	if state == "" {
		http.Redirect(w, r, redirectURL+"?error=missing_state", http.StatusSeeOther)
		return nil
	}

	if code == "" {
		http.Redirect(w, r, redirectURL+"?error=missing_auth_code", http.StatusSeeOther)
		return nil
	}

	err := api.googleService.ConnectCallback(
		r.Context(),
		state,
		code,
	)
	if err != nil {
		log.Printf("[ERROR] google service ConnectCallback: %s\n", err)
		http.Redirect(w, r, redirectURL+"?error=server_error", http.StatusSeeOther)
		return nil
	}

	http.Redirect(w, r,
		redirectURL,
		http.StatusSeeOther)

	return nil
}
