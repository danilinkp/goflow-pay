package domain

import "errors"

var (
	ErrUserNotFound          = errors.New("user not found")
	ErrUserAlreadyExists     = errors.New("user already exists")
	ErrCompanyAlreadyExists  = errors.New("company already exists")
	ErrCompanyNotFound       = errors.New("company not found")
	ErrCannotDeleteSelf      = errors.New("cannot delete yourself")
	ErrCannotDeleteLastAdmin = errors.New("cannot delete last company admin")
	ErrUserNotInCompany      = errors.New("user does not belong to company")
)
