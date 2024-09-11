package types

import "errors"

var (
	ErrOldOrder       = errors.New("order was already uploaded")
	ErrOrderExists    = errors.New("order was already uploaded by another user")
	ErrManyRequests   = errors.New("too many requests to accrual system")
	ErrOrderNotFound  = errors.New("order not found in accrual system")
	ErrOrderInProcess = errors.New("order in process")
)
