package openid

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Arup3201/torb/auth"
	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/core/users"
	"github.com/Arup3201/torb/models"
	"github.com/go-playground/validator/v10"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/gorm"
)

const (
	USERINFO_ENDPOINT = "https://openidconnect.googleapis.com/v1/userinfo"
)

func getStateKey(s string) string {
	return "AUTH_STATE:" + s
}

func getVerifierKey(s string) string {
	return "PKCE_VERIFIER:" + s
}

func getTokenKey(s string) string {
	return "TOKEN:" + s
}

// https://developers.google.com/identity/openid-connect/openid-connect#discovery
type GoogleUserInfo struct {
	Subject string `json:"sub" validate:"required"`
	Name    string `json:"name" validate:"required"`
	Email   string `json:"email" validate:"required"`
}

type GoogleService struct {
	config      *oauth2.Config
	txManager   *core.TxManager
	userRepo    *users.UserRepository
	oauthRepo   *OauthRepository
	stringStore *StringStore
}

func NewGoogleService(
	clientID, clientSecret string,
	redirectURI string,
	txManager *core.TxManager,
	userRepo *users.UserRepository,
	oauthRepo *OauthRepository,
	stringStore *StringStore,
) *GoogleService {

	conf := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{"openid", "profile", "email"},
		Endpoint:     google.Endpoint,
		RedirectURL:  redirectURI,
	}

	return &GoogleService{
		config:      conf,
		txManager:   txManager,
		userRepo:    userRepo,
		oauthRepo:   oauthRepo,
		stringStore: stringStore,
	}
}

func (s *GoogleService) GetAuthCodeURL(ctx context.Context) (string, error) {

	state, _ := auth.GetRandomToken(32)
	err := s.stringStore.Store(ctx, getStateKey(state), state, 1*time.Minute)
	if err != nil {
		return "", fmt.Errorf("string store Store: %w", err)
	}
	verifier := oauth2.GenerateVerifier()
	err = s.stringStore.Store(ctx, getVerifierKey(state), verifier, 1*time.Minute)
	if err != nil {
		return "", fmt.Errorf("string store Store: %w", err)
	}
	url := s.config.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))
	return url, nil
}

func (s *GoogleService) Callback(ctx context.Context,
	state, code string) (string, string, error) {

	var userID string
	var acc models.OauthAccount

	userInfo, err := s.getUserInfo(ctx, state, code)
	if err != nil {
		return "", "", fmt.Errorf("getUserInfo: %w", err)
	}

	acc, err = s.oauthRepo.Get(ctx, userInfo.Subject, OAUTH_PROVIDER_GOOGLE)
	if err == core.ErrNotFound {
		var username = strings.Split(userInfo.Email, "@")[0]

		err = s.txManager.WithTx(func(tx *gorm.DB) error {
			userRepo := s.userRepo.WithTx(tx)
			oauthRepo := s.oauthRepo.WithTx(tx)

			userID, err = userRepo.Create(
				ctx,
				username,
				userInfo.Email,
				"",
				&userInfo.Name,
				nil,
			)
			if err != nil {
				return fmt.Errorf("user repository create: %w", err)
			}

			err = oauthRepo.Create(
				ctx,
				userInfo.Subject,
				OAUTH_PROVIDER_GOOGLE,
				userID,
				userInfo.Email,
			)
			if err != nil {
				return fmt.Errorf("oauth account repository create: %w", err)
			}

			return nil
		})
		if err != nil {
			return "", "", err
		}

		acc, err = s.oauthRepo.Get(ctx, userInfo.Subject, OAUTH_PROVIDER_GOOGLE)
		if err != nil {
			return "", "", fmt.Errorf("oauth repository Get: %w", err)
		}

	} else if err != nil {
		return "", "", fmt.Errorf("oauth repository Get: %w", err)
	}

	token, _ := auth.GetRandomToken(32)
	err = s.stringStore.Store(ctx, getTokenKey(token), token, 1*time.Minute)
	if err != nil {
		return "", "", fmt.Errorf("string store Store: %w", err)
	}

	return acc.UserID, token, nil
}

func (s *GoogleService) getUserInfo(ctx context.Context,
	state, code string) (*GoogleUserInfo, error) {

	_, err := s.stringStore.Get(ctx, getStateKey(state))
	if err != nil {
		return nil, core.ErrInvalidValue
	}

	verifier, err := s.stringStore.Get(ctx, getVerifierKey(state))
	if err != nil {
		return nil, core.ErrInvalidValue
	}

	tok, err := s.config.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return nil, fmt.Errorf("code exchange: %w", err)
	}

	client := s.config.Client(ctx, tok)
	res, err := client.Get(USERINFO_ENDPOINT)
	if err != nil {
		return nil, fmt.Errorf("client get: %w", err)
	}
	defer res.Body.Close()

	var userInfo GoogleUserInfo
	if err := json.NewDecoder(res.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("userInfo decode: %w", core.ErrInvalidValue)
	}
	if err := validator.New().Struct(userInfo); err != nil {
		return nil, fmt.Errorf("userInfo validate: %w", core.ErrInvalidValue)
	}

	return &userInfo, nil
}

func (s *GoogleService) ValidToken(ctx context.Context,
	token string) error {

	_, err := s.stringStore.Get(ctx, getTokenKey(token))
	if err != nil {
		return core.ErrInvalidValue
	}

	return nil
}

func (s *GoogleService) HasAccount(ctx context.Context,
	id string) (bool, error) {

	hasAccount, err := s.oauthRepo.Exists(ctx, id)
	if err != nil {
		return false, err
	}

	return hasAccount, nil
}

func (s *GoogleService) CallbackForExistingUser(ctx context.Context,
	state, code string) error {

	userInfo, err := s.getUserInfo(ctx, state, code)
	if err != nil {
		return fmt.Errorf("get user information in callback: %w", err)
	}

	user, err := s.userRepo.GetByEmail(ctx, userInfo.Email)
	if err != nil {
		return fmt.Errorf("get user by email: %w", err)
	}

	err = s.oauthRepo.Create(
		ctx,
		userInfo.Subject,
		OAUTH_PROVIDER_GOOGLE,
		user.ID,
		user.Email,
	)
	if err != nil {
		return fmt.Errorf("create oauth account in database: %w", err)
	}

	return nil
}
