package manual

import (
	"context"
	"fmt"
	"time"

	"github.com/Arup3201/torb/auth"
	"github.com/Arup3201/torb/core"
	"golang.org/x/crypto/bcrypt"
)

const (
	RESET_TOKEN_DURATION_DEFAULT = 24 * time.Hour
)

type PasswordService struct {
	accountRepo *ManualAccountRepository

	ResetTokenDuration   time.Duration
	PasswordGenerateCost int
}

func NewPasswordService(accountRepo *ManualAccountRepository) *PasswordService {

	return &PasswordService{
		accountRepo: accountRepo,

		ResetTokenDuration:   RESET_TOKEN_DURATION_DEFAULT,
		PasswordGenerateCost: PASSWORD_GENERATION_COST_DEFAULT,
	}
}

/*
Get password reset token.

Usage:

Use the token with an URL and send it as email to a verified
email address. The user can reset his/her password by going
to the URL.
*/
func (s *PasswordService) GetResetToken(ctx context.Context,
	email string) (string, error) {

	acc, err := s.accountRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", fmt.Errorf("manual account repository get by email: %w", err)
	}

	if !acc.EmailVerified {
		return "", fmt.Errorf("email not yet verified: %w", core.ErrInvalidValue)
	}

	token, err := auth.GetRandomToken(32)
	if err != nil {
		return "", fmt.Errorf("get random token: %w", err)
	}

	tokenSHA := auth.GetTokenSHA(token)
	tokenExpiresAt := time.Now().UTC().Add(s.ResetTokenDuration)
	err = s.accountRepo.UpdateResetPasswordToken(ctx,
		acc,
		tokenSHA,
		tokenExpiresAt)
	if err != nil {
		return "", fmt.Errorf("account repository update reset password token: %w", err)
	}

	return token, nil
}

/*
Reset password.

Takes the new password and the token received from GetResetToken
method.

Usage:

After user goes to the reset password page and sends their new password
along with the token, this function will update the password after
verifying the token.

NOTE:

It validates if the reset token has been expired or already used.
If so then returns invalid value error otherwise proceeds to update
the previous password. Just like register, it saves the hashed
password generated with bcrypt.
*/
func (s *PasswordService) Reset(ctx context.Context,
	token, password string) error {

	tokenSHA := auth.GetTokenSHA(token)
	acc, err := s.accountRepo.GetByResetToken(ctx, tokenSHA)
	if err != nil {
		return fmt.Errorf("manual account repository get by verification token: %w", err)
	}

	now := time.Now().UTC()
	if acc.ResetPasswordTokenExpiresAt.Sub(now) < 0 {
		return fmt.Errorf("reset token has expired: %w", core.ErrInvalidValue)
	}

	if acc.ResetPasswordTokenUsedAt != nil {
		return fmt.Errorf("reset token already used: %w", core.ErrInvalidValue)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), s.PasswordGenerateCost)
	if err != nil {
		return fmt.Errorf("bcrypt generate from password: %w", err)
	}

	err = s.accountRepo.UpdatePassword(ctx, acc, hashedPassword)
	if err != nil {
		return fmt.Errorf("account repository update password: %w", err)
	}

	return nil
}
