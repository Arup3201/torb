package api

import (
	"errors"
	"net/http"

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
