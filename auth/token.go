package auth

import (
	"context"
	"crypto/rsa"
	"fmt"
	"time"
)

const (
	ACCESS_TOKEN_DURATION_DEFAULT  = 15 * time.Minute
	REFRESH_TOKEN_DURATION_DEFAULT = 7 * 24 * time.Hour
)

type Token struct {
	Value     string
	ExpiresAt time.Time
}

type TokenService struct {
	store      *TokenStore
	issuer     string
	privateKey *rsa.PrivateKey

	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration
}

func NewTokenService(store *TokenStore,
	issuer string,
	privateKey *rsa.PrivateKey) *TokenService {
	return &TokenService{
		store:      store,
		issuer:     issuer,
		privateKey: privateKey,

		AccessTokenDuration:  ACCESS_TOKEN_DURATION_DEFAULT,
		RefreshTokenDuration: REFRESH_TOKEN_DURATION_DEFAULT,
	}
}

func (s *TokenService) CreateAccessToken(ctx context.Context,
	userID string) (*Token, error) {

	accessExpiry := time.Now().Add(s.AccessTokenDuration)
	accessClaims := NewClaims(s.issuer, userID, accessExpiry)
	jwt, err := JWTFromClaims(accessClaims, JWT_ALG_RSA)
	if err != nil {
		return nil, fmt.Errorf("jwt from claims: %w", err)
	}

	accessToken, err := jwt.Sign(s.privateKey)
	if err != nil {
		return nil, fmt.Errorf("jwt sign: %w", err)
	}

	return &Token{
		Value:     accessToken,
		ExpiresAt: accessExpiry,
	}, nil
}

func (s *TokenService) CreateRefreshToken(ctx context.Context,
	userID string) (*Token, error) {

	refreshExpiry := time.Now().Add(s.RefreshTokenDuration)
	refreshClaims := NewClaims(s.issuer, userID, refreshExpiry)

	s.store.Save(ctx, refreshClaims.Jti, refreshExpiry)

	jwt, err := JWTFromClaims(refreshClaims, JWT_ALG_RSA)
	if err != nil {
		return nil, fmt.Errorf("jwt from claims: %w", err)
	}

	refreshToken, err := jwt.Sign(s.privateKey)
	if err != nil {
		return nil, fmt.Errorf("jwt sign: %w", err)
	}

	return &Token{
		Value:     refreshToken,
		ExpiresAt: refreshExpiry,
	}, nil
}

// Get user ID from refresh token
func (s *TokenService) GetUserID(ctx context.Context,
	token string) (string, error) {

	claims, err := ClaimsFromToken(token, &s.privateKey.PublicKey)
	if err != nil {
		return "", fmt.Errorf("claims from token: %w", err)
	}

	return claims.Subject, nil
}

func (s *TokenService) RevokeRefreshToken(ctx context.Context,
	token string) error {

	claims, err := ClaimsFromToken(token, &s.privateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("claims from token: %w", err)
	}

	err = s.store.Revoke(ctx, claims.Jti)
	if err != nil {
		return fmt.Errorf("token store revoke refresh token: %w", err)
	}

	return nil
}
