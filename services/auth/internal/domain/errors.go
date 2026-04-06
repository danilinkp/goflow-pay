package domain

import "errors"

var (
	ErrUserNotFound         = errors.New("user not found")
	ErrUserAlreadyExists    = errors.New("user already exists")
	ErrCompanyAlreadyExists = errors.New("company already exists")
	ErrCompanyNotFound      = errors.New("company not found")
)
