package model

import (
	"errors"
	"time"
)

var (
	ErrOldOrder       = errors.New("order was already uploaded")
	ErrOrderExists    = errors.New("order was already uploaded by another user")
	ErrManyRequests   = errors.New("too many requests to accrual system")
	ErrOrderNotFound  = errors.New("order not found in accrual system")
	ErrOrderInProcess = errors.New("order in process")
)

type User struct {
	ID    int
	Login string
	Hash  []byte
}

type Order struct {
	ID         int       `json:"number,string"`
	Status     string    `json:"status"`
	UploadedAt time.Time `json:"uploaded_at"`
	Accrual    *float64  `json:"accrual,omitempty"`
}
