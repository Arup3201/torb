package core

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type CacheManager struct {
	redis *redis.Client
}

func NewCacheManager(redis *redis.Client) *CacheManager {
	return &CacheManager{
		redis: redis,
	}
}

func (m *CacheManager) CacheKeyFromURLKey(
	key string,
) string {

	return "torb:cache:" +
		key
}

func (m *CacheManager) Set(
	ctx context.Context,
	key string,
	value any,
	ttl time.Duration,
) error {

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal cache value: %w", err)
	}

	err = m.redis.Set(ctx, key, data, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to store cache value: %w", err)
	}

	return nil
}

func (m *CacheManager) Get(
	ctx context.Context,
	key string,
	destination any,
) (bool, error) {

	data, err := m.redis.Get(ctx, key).Bytes()

	if err != nil {
		if err == redis.Nil {
			return false, nil
		}

		return false, fmt.Errorf("failed to get cache value: %w", err)
	}

	err = json.Unmarshal(data, destination)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal cache value: %w", err)
	}

	return true, nil
}

func (m *CacheManager) Delete(
	ctx context.Context,
	keys ...string,
) error {

	if len(keys) == 0 {
		return nil
	}

	err := m.redis.Del(ctx, keys...).Err()
	if err != nil {
		return fmt.Errorf("failed to delete cache: %w", err)
	}

	return nil
}
