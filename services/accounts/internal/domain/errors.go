package domain

import "errors"

var (
	ErrAccountNotFound               = errors.New("account not found")
	ErrAccountAlreadyExists          = errors.New("account already exists")
	ErrBankAccountAlreadyExists      = errors.New("bank account already exists")
	ErrBankAccountNotFound           = errors.New("bank account not found")
	ErrAccountOperationNotFound      = errors.New("account operation not found")
	ErrAccountOperationAlreadyExists = errors.New("account operation already exists")
	ErrBankOperationAlreadyExists    = errors.New("bank operation already exists")
	ErrBankOperationNotFound         = errors.New("bank operation not found")
	ErrStatementAlreadyExists        = errors.New("statement already exists")
	ErrStatementNotFound             = errors.New("statement not found")
)
