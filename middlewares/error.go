package middlewares

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/Arup3201/torb/core"
)

const RESPONSE_ERROR_STATUS = "error"

type ErrorBody struct {
	Id      string `json:"id"`
	Message string `json:"message"`
}

type HTTPErrorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type HTTPErrorHandler func(w http.ResponseWriter, r *http.Request) error

func (fn HTTPErrorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := fn(w, r); err != nil {
		fmt.Printf("[ERROR] %s\n", err)

		if errors.Is(err, core.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(HTTPErrorResponse{
				Status:  RESPONSE_ERROR_STATUS,
				Message: "Resource not found",
			})
		} else if errors.Is(err, core.ErrUnauthorized) {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(HTTPErrorResponse{
				Status:  RESPONSE_ERROR_STATUS,
				Message: "User is not authorized",
			})
		} else if errors.Is(err, core.ErrForbidden) {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(HTTPErrorResponse{
				Status:  RESPONSE_ERROR_STATUS,
				Message: "Resource or action is forbidden for the user",
			})
		} else if errors.Is(err, core.ErrInvalidValue) {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(HTTPErrorResponse{
				Status:  RESPONSE_ERROR_STATUS,
				Message: "Request payload is incorrect",
			})
		} else if errors.Is(err, core.ErrDuplicate) {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(HTTPErrorResponse{
				Status:  RESPONSE_ERROR_STATUS,
				Message: "Duplicate resource found while processing the request",
			})
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(HTTPErrorResponse{
				Status:  RESPONSE_ERROR_STATUS,
				Message: "Server encountered an error",
			})
		}

		// Already wrote the response, make sure we don't overwrite the response
		return // TODO: May create a bug later!!
	}
}
