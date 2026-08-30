package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
	"time"

	"github.com/Arup3201/torb/auth"
	"github.com/Arup3201/torb/auth/manual"
	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/core/users"
	"github.com/go-playground/validator/v10"
	"github.com/resend/resend-go/v3"
)

type RegisterRequest struct {
	Username    string  `json:"username" validate:"required"`
	Email       string  `json:"email" validate:"required"`
	DisplayName *string `json:"display_name"`
	Password    string  `json:"password" validate:"required"`
}

type VerifyEmailRequest struct {
	Token string `json:"token" validate:"required"`
}

type ResendVerificationRequest struct {
	Email string `json:"email" validate:"required"`
}

type PasswordResetEmailRequest struct {
	Email string `json:"email" validate:"required"`
}

type PasswordResetRequest struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	AccessToken string      `json:"access_token"`
	ExpiresAt   time.Time   `json:"expires_at"`
	User        core.Avatar `json:"user"`
}

type AuthApi struct {
	registerService *manual.RegisterService
	tokenService    *auth.TokenService
	emailService    *manual.EmailService
	passwordService *manual.PasswordService
	userService     *users.UserService
	emailClient     *resend.Client

	FrontendVerifyURL        string
	FrontendPasswordResetURL string
}

func NewAuthApi(
	registerService *manual.RegisterService,
	tokenService *auth.TokenService,
	emailService *manual.EmailService,
	passwordService *manual.PasswordService,
	userService *users.UserService,
	emailClient *resend.Client,
	verifyURL string,
	resetURL string,
) *AuthApi {
	return &AuthApi{
		registerService:          registerService,
		tokenService:             tokenService,
		passwordService:          passwordService,
		emailService:             emailService,
		userService:              userService,
		emailClient:              emailClient,
		FrontendVerifyURL:        verifyURL,
		FrontendPasswordResetURL: resetURL,
	}
}

func (api *AuthApi) SendVerificationEmail(ctx context.Context, token, email string) error {

	verificationLink := fmt.Sprintf("%s?token=%s", api.FrontendVerifyURL, token)
	content := fmt.Sprintf(`
	Hello %s, 
	Here is your email verification link: 
	<a href='%s'>Click to verify</a>
	`,
		email,
		verificationLink)

	params := &resend.SendEmailRequest{
		From:    "Arup <hello@contact.itsdeployedbyme.dpdns.org>",
		To:      []string{email},
		Html:    content,
		Subject: "Email verification",
		ReplyTo: "hello@contact.itsdeployedbyme.dpdns.org",
	}

	sent, err := api.emailClient.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("email client send: %w", err)
	}

	log.Printf("[INFO] Email sent: %s\n", sent.Id)

	return nil
}

func (api *AuthApi) Register(w http.ResponseWriter, r *http.Request) error {

	var payload RegisterRequest

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&payload); err != nil {
		return fmt.Errorf("decoder decode: %w: %w", err, core.ErrInvalidValue)
	}
	if err := validator.New().Struct(payload); err != nil {
		return fmt.Errorf("validator new: %w: %w", err, core.ErrInvalidValue)
	}

	userID, err := api.registerService.CreateAccount(
		r.Context(),
		payload.Username,
		payload.Email,
		payload.Password,
		payload.DisplayName,
	)
	if err != nil {
		return fmt.Errorf("register service create account: %w", err)
	}

	token, err := api.emailService.GetVerificationToken(
		r.Context(),
		userID)
	if err != nil {
		return fmt.Errorf("email service get verification token: %w", err)
	}

	err = api.SendVerificationEmail(r.Context(), token, payload.Email)
	if err != nil {
		return fmt.Errorf("send verification email: %w", err)
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(HTTPSuccessResponse[any]{
		Status:  RESPONSE_SUCCESS_STATUS,
		Message: "Registration successful",
	})

	return nil
}

func (api *AuthApi) VerifyEmail(w http.ResponseWriter, r *http.Request) error {

	var payload VerifyEmailRequest

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&payload); err != nil {
		return fmt.Errorf("decoder decode: %w", core.ErrInvalidValue)
	}
	if err := validator.New().Struct(payload); err != nil {
		return fmt.Errorf("validator new: %w", core.ErrInvalidValue)
	}

	err := api.emailService.Verify(
		r.Context(),
		payload.Token,
	)
	if err != nil {
		return fmt.Errorf("email service verify: %w", err)
	}

	json.NewEncoder(w).Encode(HTTPSuccessResponse[any]{
		Status:  RESPONSE_SUCCESS_STATUS,
		Message: "Email verification successful",
	})

	return nil
}

func (api *AuthApi) ResendVerificationEmail(w http.ResponseWriter, r *http.Request) error {

	var payload ResendVerificationRequest

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&payload); err != nil {
		return fmt.Errorf("decoder decode: %w", core.ErrInvalidValue)
	}
	if err := validator.New().Struct(payload); err != nil {
		return fmt.Errorf("validator new: %w", core.ErrInvalidValue)
	}

	token, err := api.emailService.GetVerificationTokenForEmail(
		r.Context(),
		payload.Email,
	)
	if err != nil {
		return fmt.Errorf("email service get verify token for email: %w", err)
	}

	err = api.SendVerificationEmail(r.Context(), token, payload.Email)
	if err != nil {
		return fmt.Errorf("send verification email: %w", err)
	}

	json.NewEncoder(w).Encode(HTTPSuccessResponse[any]{
		Status:  RESPONSE_SUCCESS_STATUS,
		Message: "Email verification token sent again.",
	})

	return nil
}

func (api *AuthApi) SendPasswordResetEmail(w http.ResponseWriter, r *http.Request) error {

	var payload PasswordResetEmailRequest

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&payload); err != nil {
		return fmt.Errorf("decoder decode: %w", core.ErrInvalidValue)
	}
	if err := validator.New().Struct(payload); err != nil {
		return fmt.Errorf("validator new: %w", core.ErrInvalidValue)
	}

	token, err := api.passwordService.GetResetToken(
		r.Context(),
		payload.Email,
	)
	if err != nil {
		return fmt.Errorf("password service get reset token: %w", err)
	}

	verificationLink := fmt.Sprintf("%s?token=%s", api.FrontendPasswordResetURL, token)
	content := fmt.Sprintf(`
	Hello %s, 
	Here is your password reset link: 
	<a href='%s'>Click to verify</a>
	`,
		payload.Email,
		verificationLink)

	params := &resend.SendEmailRequest{
		From:    "Arup <hello@contact.itsdeployedbyme.dpdns.org>",
		To:      []string{payload.Email},
		Html:    content,
		Subject: "Reset Password",
		ReplyTo: "hello@contact.itsdeployedbyme.dpdns.org",
	}

	sent, err := api.emailClient.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("email client send: %w", err)
	}

	log.Printf("[INFO] Email sent: %s\n", sent.Id)

	json.NewEncoder(w).Encode(HTTPSuccessResponse[any]{
		Status:  RESPONSE_SUCCESS_STATUS,
		Message: "Reset password email sent successfully.",
	})

	return nil
}

func (api *AuthApi) ResetPassword(w http.ResponseWriter, r *http.Request) error {

	var payload PasswordResetRequest

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&payload); err != nil {
		return fmt.Errorf("decoder decode: %w", core.ErrInvalidValue)
	}
	if err := validator.New().Struct(payload); err != nil {
		return fmt.Errorf("validator new: %w", core.ErrInvalidValue)
	}

	err := api.passwordService.Reset(
		r.Context(),
		payload.Token,
		payload.Password,
	)
	if err != nil {
		return fmt.Errorf("password service reset: %w", err)
	}

	json.NewEncoder(w).Encode(HTTPSuccessResponse[any]{
		Status:  RESPONSE_SUCCESS_STATUS,
		Message: "Password reset is successful.",
	})

	return nil
}

func (api *AuthApi) Login(w http.ResponseWriter, r *http.Request) error {

	var payload LoginRequest

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&payload); err != nil {
		return fmt.Errorf("decoder decode: %w", core.ErrInvalidValue)
	}
	if err := validator.New().Struct(payload); err != nil {
		return fmt.Errorf("validator new: %w", core.ErrInvalidValue)
	}

	userID, err := api.registerService.GetUserID(
		r.Context(),
		payload.Email,
		payload.Password,
	)
	if err != nil {
		return fmt.Errorf("register service get user ID: %w", err)
	}

	refreshToken, err := api.tokenService.CreateRefreshToken(
		r.Context(),
		userID,
	)
	if err != nil {
		return fmt.Errorf("token service create refresh token: %w", err)
	}

	accessToken, err := api.tokenService.CreateAccessToken(
		r.Context(),
		userID,
	)
	if err != nil {
		return fmt.Errorf("token service create access token: %w", err)
	}

	user, err := api.userService.Get(
		r.Context(),
		userID)
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
		SameSite: http.SameSiteNoneMode,
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

func (api *AuthApi) Refresh(w http.ResponseWriter, r *http.Request) error {

	cookies := r.Cookies()
	tInd := slices.IndexFunc(cookies, func(c *http.Cookie) bool {
		return c.Name == REFRESH_TOKEN_COOKIE_NAME
	})
	if tInd == -1 {
		return fmt.Errorf("refresh token missing: %w", core.ErrUnauthorized)
	}

	refreshTokenStr := cookies[tInd].Value
	if refreshTokenStr == "" {
		return fmt.Errorf("refresh token empty: %w", core.ErrUnauthorized)
	}

	userID, err := api.tokenService.GetUserID(
		r.Context(),
		refreshTokenStr,
	)
	if err != nil {
		return fmt.Errorf("token service get user ID: %w", err)
	}

	err = api.tokenService.RevokeRefreshToken(
		r.Context(),
		refreshTokenStr,
	)
	if err != nil {
		return fmt.Errorf("token service revoke refresh token: %w", err)
	}

	refreshToken, err := api.tokenService.CreateRefreshToken(
		r.Context(),
		userID,
	)
	if err != nil {
		return fmt.Errorf("token service create refresh token: %w", err)
	}

	accessToken, err := api.tokenService.CreateAccessToken(
		r.Context(),
		userID,
	)
	if err != nil {
		return fmt.Errorf("token service create access token: %w", err)
	}

	user, err := api.userService.Get(
		r.Context(),
		userID)
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
		SameSite: http.SameSiteNoneMode,
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

func (api *AuthApi) Logout(w http.ResponseWriter, r *http.Request) error {

	cookies := r.Cookies()
	tInd := slices.IndexFunc(cookies, func(c *http.Cookie) bool {
		return c.Name == REFRESH_TOKEN_COOKIE_NAME
	})
	if tInd == -1 {
		return fmt.Errorf("refresh token missing: %w", core.ErrUnauthorized)
	}

	refreshTokenStr := cookies[tInd].Value
	if refreshTokenStr == "" {
		return fmt.Errorf("refresh token empty: %w", core.ErrUnauthorized)
	}

	err := api.tokenService.RevokeRefreshToken(
		r.Context(),
		refreshTokenStr,
	)
	if err != nil {
		return fmt.Errorf("token service revoke refresh token: %w", err)
	}

	cookie := &http.Cookie{
		Name:     REFRESH_TOKEN_COOKIE_NAME,
		Value:    "",
		Path:     "/", // TODO: auth path only
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
	}
	http.SetCookie(w, cookie)

	json.NewEncoder(w).Encode(HTTPSuccessResponse[any]{
		Message: "Log out successful",
		Status:  RESPONSE_SUCCESS_STATUS,
	})

	return nil
}
