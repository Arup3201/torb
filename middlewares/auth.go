package middlewares

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Arup3201/torb/auth"
	"github.com/Arup3201/torb/core"
)

type Authenticator struct {
	tokenService *auth.TokenService
}

func NewAuthenticator(tokenService *auth.TokenService) *Authenticator {
	return &Authenticator{
		tokenService: tokenService,
	}
}

func (m *Authenticator) IsAuthenticated(next HTTPErrorHandler) HTTPErrorHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		bearer := r.Header.Get("Authorization")
		if strings.Trim(bearer, " ") == "" {
			return fmt.Errorf("Authorization token missing: %w", core.ErrUnauthorized)
		}
		bearerToken := strings.Fields(bearer)
		if bearerToken[0] != "Bearer" {
			return fmt.Errorf("Malformed authorization token: %w", core.ErrUnauthorized)
		}

		userID, err := m.tokenService.GetUserID(
			r.Context(),
			bearerToken[1],
		)
		if err != nil {
			return fmt.Errorf("Failed to get claims from token: %w", core.ErrUnauthorized)
		}

		reqWithCtx := r.WithContext(NewContext(r.Context(), userID))

		next.ServeHTTP(w, reqWithCtx)

		return nil
	}
}
