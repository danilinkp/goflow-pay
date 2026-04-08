package domain

import "errors"

var (
	ErrNotificationNotFound      = errors.New("notification not found")
	ErrNotificationAlreadyExists = errors.New("notification already exists")
	ErrCachedCompanyNotFound     = errors.New("cached company not found")
)
