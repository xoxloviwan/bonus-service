package model

import (
	"errors"
	"fmt"
	"strings"
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
	ID         int         `json:"number,string"`
	Status     OrderStatus `json:"status"`
	UploadedAt time.Time   `json:"uploaded_at"`
	Accrual    *float64    `json:"accrual,omitempty"`
}

type AccrualResp struct {
	Order   int         `json:"order,string"`
	Status  OrderStatus `json:"status"`
	Accrual *float64    `json:"accrual,omitempty"`
}

//go:generate stringer -type=OrderStatus --trimprefix OrderStatus
type OrderStatus int

const (
	OrderStatusNew OrderStatus = iota
	OrderStatusRegistered
	OrderStatusProcessing
	OrderStatusProcessed
	OrderStatusInvalid
)

// MarshalText implements the encoding.TextMarshaler interface.
func (s OrderStatus) MarshalText() ([]byte, error) {
	return []byte(strings.ToUpper(s.String())), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (s *OrderStatus) UnmarshalText(b []byte) error {
	switch string(b) {
	case "REGISTERED":
		*s = OrderStatusRegistered
		return nil
	case "INVALID":
		*s = OrderStatusInvalid
		return nil
	case "PROCESSING":
		*s = OrderStatusProcessing
		return nil
	case "PROCESSED":
		*s = OrderStatusProcessed
		return nil
	}
	return fmt.Errorf("invalid order status: %s", b)

}
