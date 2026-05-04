package api

type HTTPSuccessResponse[T any] struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Data    *T     `json:"data,omitempty"`
}
