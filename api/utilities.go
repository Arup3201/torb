package api

import (
	"errors"
	"net/http"
)

func GetUserID(req *http.Request) (string, error) {
	ctx := req.Context()
	userId, ok := ctx.Value(CTX_USER_KEY).(string)
	if !ok || userId == "" {
		return "", errors.New("empty context")
	}
	return userId, nil
}
