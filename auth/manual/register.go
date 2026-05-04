package manual

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/core/users"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	PASSWORD_GENERATION_COST_DEFAULT = 14
)

var emailRegex, _ = regexp.Compile(
	`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`,
)

type RegisterService struct {
	txManager  *core.TxManager
	manualRepo *ManualAccountRepository
	userRepo   *users.UserRepository

	// bcrypt.GenerateFromPassword cost value
	// Reference
	// https://pkg.go.dev/golang.org/x/crypto/bcrypt#GenerateFromPassword
	PasswordGenerateCost int
}

func NewRegisterService(
	txManager *core.TxManager,
	manualRepo *ManualAccountRepository,
	userRepo *users.UserRepository) *RegisterService {
	return &RegisterService{
		txManager:            txManager,
		manualRepo:           manualRepo,
		userRepo:             userRepo,
		PasswordGenerateCost: PASSWORD_GENERATION_COST_DEFAULT,
	}
}

func (s *RegisterService) CreateAccount(ctx context.Context,
	username, email, password string,
	displayName *string) (string, error) {

	var err error

	if strings.Trim(email, " ") == "" {
		return "", fmt.Errorf("email is empty: %w", core.ErrInvalidValue)
	}

	if match := emailRegex.Find([]byte(email)); match == nil {
		return "", fmt.Errorf("email pattern not supported: %w", core.ErrInvalidValue)
	}

	if strings.Trim(username, " ") == "" {
		return "", fmt.Errorf("username is empty: %w", core.ErrInvalidValue)
	}

	if strings.Trim(password, " ") == "" {
		return "", fmt.Errorf("password is empty: %w", core.ErrInvalidValue)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		s.PasswordGenerateCost)
	if err != nil {
		return "", fmt.Errorf("bcrypt generate from password: %w", err)
	}

	var userID string
	err = s.txManager.WithTx(func(tx *gorm.DB) error {
		userRepo := s.userRepo.WithTx(tx)
		manualRepo := s.manualRepo.WithTx(tx)

		userID, err = userRepo.Create(ctx, username, email, displayName, nil)
		if err != nil {
			return fmt.Errorf("user repository create: %w", err)
		}

		err = manualRepo.Create(ctx, userID, email, hashedPassword, false)
		if err != nil {
			return fmt.Errorf("manual account repository create: %w", err)
		}

		return nil
	})
	if err != nil {
		return "", fmt.Errorf("transaction: %w", err)
	}

	return userID, nil
}

func (s *RegisterService) GetUserID(ctx context.Context,
	email, password string) (string, error) {

	account, err := s.manualRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", fmt.Errorf("account repository get by email: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword(account.PasswordHash, []byte(password)); err != nil {
		return "", fmt.Errorf("password mismatch: %w", core.ErrInvalidValue)
	}

	if !account.EmailVerified {
		return "", fmt.Errorf("email verification is pending: %w", core.ErrInvalidValue)
	}

	return account.UserID, nil
}
