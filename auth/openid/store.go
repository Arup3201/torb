package openid

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type StringStore struct {
	client *redis.Client
}

func NewStringStore(client *redis.Client) *StringStore {
	return &StringStore{
		client: client,
	}
}

func (s *StringStore) Store(ctx context.Context,
	key, value string,
	exp time.Duration) error {

	err := s.client.Set(ctx, key, value, exp).Err()
	if err != nil {
		return fmt.Errorf("redis set: %w", err)
	}

	return nil
}

func (s *StringStore) Get(ctx context.Context,
	key string) (string, error) {

	res, err := s.client.Get(ctx, key).Result()
	if err != nil {
		return "", fmt.Errorf("redis get: %w", err)
	}

	return res, nil
}
