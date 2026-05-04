package core

import "errors"

var ErrUnauthorized = errors.New("unauthorized")
var ErrNotFound = errors.New("resource not found")
var ErrDuplicate = errors.New("duplicate value")
var ErrInvalidValue = errors.New("invalid value")
var ErrForbidden = errors.New("forbidden")
