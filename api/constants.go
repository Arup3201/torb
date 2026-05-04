package api

const (
	RESPONSE_SUCCESS_STATUS = "success"
	RESPONSE_ERROR_STATUS   = "error"
)

// HTTP Error ID
const (
	ERR_UNAUTHORIZED       = "unauthorized"
	ERR_INVALID_QUERY      = "invalid_query"
	ERR_INVALID_BODY       = "invalid_body"
	ERR_ACCESS_DENIED      = "access_denied"
	ERR_RESOURCE_NOT_FOUND = "resource_not_found"
	ERR_SERVER_ERROR       = "server_error"
)

const (
	CTX_USER_KEY = "USER_ID"
)

const (
	REFRESH_TOKEN_COOKIE_NAME = "TOKEN"
)
