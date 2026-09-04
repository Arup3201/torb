package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Arup3201/torb/core"
	"github.com/Arup3201/torb/middlewares"
)

func GetUserID(req *http.Request) (string, error) {
	ctx := req.Context()
	userId, ok := ctx.Value(middlewares.CTX_USER_KEY).(string)
	if !ok || userId == "" {
		return "", errors.New("empty context")
	}
	return userId, nil
}

func CheckCacheHit(
	ctx context.Context,
	cacheManager *core.CacheManager,
	w http.ResponseWriter,
	key string,
	dest any) bool {
	cacheKey := cacheManager.CacheKeyFromURLKey(key)
	hit, err := cacheManager.Get(ctx, cacheKey, dest)
	if err != nil {
		fmt.Printf("[ERROR] Cache Manager: %s\n", err)
	}

	if hit {
		json.NewEncoder(w).Encode(dest)
	}

	return hit
}

func CacheResponse(
	ctx context.Context,
	cacheManager *core.CacheManager,
	w http.ResponseWriter,
	key string,
	response any,
	cacheDuration time.Duration,
) {

	err := cacheManager.Set(
		ctx,
		cacheManager.CacheKeyFromURLKey(key),
		response,
		cacheDuration)
	if err != nil {
		fmt.Printf("[ERROR] Cache Manager: %s\n", err)
	}

	json.NewEncoder(w).Encode(response)
}
