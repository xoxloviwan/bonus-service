package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"gophermart/internal/types"

	"github.com/theplant/luhn"
)

type Store interface {
	AddUser(ctx context.Context, login string, hash []byte) (int, error)
	GetUser(ctx context.Context, login string) ([]byte, int, error)
	AddOrder(ctx context.Context, orderID int, userID int) error
}

type Poller interface {
	Push(orderID int)
}

type Handler struct {
	store  Store
	poller Poller
}

// borrowed from benchfmt/internal/bytesconv/atoi.go
// atoi is equivalent to ParseInt(s, 10, 0), converted to type int.
func atoi(s []byte) (int, error) {
	const intSize = 32 << (^uint(0) >> 63)

	sLen := len(s)
	if intSize == 32 && (0 < sLen && sLen < 10) ||
		intSize == 64 && (0 < sLen && sLen < 19) {
		// Fast path for small integers that fit int type.
		s0 := s

		n := 0
		for _, ch := range s {
			ch -= '0'
			if ch > 9 {
				return 0, fmt.Errorf("atoi: invalid bytes: %q", string(s0))
			}
			n = n*10 + int(ch)
		}

		return n, nil
	}
	return 0, errors.New("atoi: not realized")
}

func (h *Handler) NewOrder(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	orderID, err := atoi(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !luhn.Valid(orderID) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	userID := r.Context().Value(userIDCtxKey{}).(int)
	err = h.store.AddOrder(r.Context(), orderID, userID)
	if err != nil {
		if errors.Is(err, types.ErrOldOrder) {
			w.WriteHeader(http.StatusOK)
			return
		}
		if errors.Is(err, types.ErrOrderExists) {
			w.WriteHeader(http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.poller.Push(orderID)
	w.WriteHeader(http.StatusAccepted)
}

func NewHandler(store Store) *Handler {
	return &Handler{store}
}

type BalanceResponse struct {
	Total   float64 `json:"current"`
	Debited float64 `json:"withdrawn"`
}

func (h *Handler) Balance(w http.ResponseWriter, r *http.Request) {
	fakeBalance := BalanceResponse{ //TODO
		Total:   1000,
		Debited: 0,
	}
	w.Header().Set("Content-Type", "application/json")
	resp, err := json.Marshal(fakeBalance)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(resp)
}
