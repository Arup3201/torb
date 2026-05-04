package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/Arup3201/torb/core"
	"github.com/redis/go-redis/v9"
)

func getTokenKey(id string) string {
	return "token:" + id
}

type TokenStore struct {
	redis *redis.Client
}

func NewTokenStore(redis *redis.Client) *TokenStore {
	return &TokenStore{
		redis: redis,
	}
}

func (s *TokenStore) Save(ctx context.Context,
	id string, expiresAt time.Time) error {

	var err error
	var key string

	key = getTokenKey(id)
	err = s.redis.HSetNX(ctx, key, "revoked", false).Err()
	if err != nil {
		return fmt.Errorf("redis hset: %w", err)
	}

	err = s.redis.ExpireAt(ctx, key, expiresAt).Err()
	if err != nil {
		return fmt.Errorf("redis expires at: %w", err)
	}

	return nil
}

func (s *TokenStore) Revoke(ctx context.Context,
	id string) error {

	var err error
	var key string

	key = getTokenKey(id)

	var revoked bool
	revoked, err = s.redis.HGet(ctx, key, "revoked").Bool()
	if err != nil {
		return fmt.Errorf("redis hget: %w", err)
	}

	if revoked {
		return fmt.Errorf("token already revoked: %w", core.ErrInvalidValue)
	}

	err = s.redis.HSet(ctx, key, "revoked", true).Err()
	if err != nil {
		return fmt.Errorf("redis hset: %w", err)
	}

	return nil
}
