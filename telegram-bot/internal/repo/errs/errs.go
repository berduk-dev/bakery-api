package errs

import "errors"

var (
	ErrTelegramIDAlreadySet = errors.New("telegram_id already assigned")
	ErrPrizeNotFound        = errors.New("prize not found")
	ErrUserAlreadyExists    = errors.New("user already exists")
	ErrPhoneAlreadyExists   = errors.New("phone already exists")
)
