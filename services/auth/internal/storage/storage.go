package storage

import "errors"

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrCompanyNotFound = errors.New("company not found")
)
