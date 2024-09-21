package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"gophermart/internal/helpers"
	"gophermart/internal/model"

	"github.com/theplant/luhn"
)

type User = model.User
type Order = model.Order

//go:generate mockgen -destination ../mock/store_mock.go -package mock gophermart/internal/api Store
type Store interface {
	AddUser(ctx context.Context, u User) (int, error)
	GetUser(ctx context.Context, login string) (User, error)
	AddOrder(ctx context.Context, orderID int, userID int) (string, error)
	ListOrders(ctx context.Context, userID int) ([]Order, error)
}

//go:generate mockgen -destination ./poller_mock.go -package api gophermart/internal/api Poller
type Poller interface {
	Push(orderID int)
}

type Handler struct {
	store  Store
	poller Poller
}

func NewHandler(store Store, poller Poller) *Handler {
	return &Handler{store, poller}
}

func (h *Handler) NewOrder(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	orderID, err := helpers.Atoi(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !luhn.Valid(orderID) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	userID := r.Context().Value(userIDCtxKey{}).(int)
	var status string
	status, err = h.store.AddOrder(r.Context(), orderID, userID)
	if err != nil {
		if errors.Is(err, model.ErrOldOrder) {
			if status != "PROCESSED" && status != "INVALID" {
				h.poller.Push(orderID)
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		if errors.Is(err, model.ErrOrderExists) {
			w.WriteHeader(http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.poller.Push(orderID)
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) OrderList(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(userIDCtxKey{}).(int)
	orders, err := h.store.ListOrders(r.Context(), userID)
	slog.Debug(fmt.Sprintf("User %d orders: %+v", userID, orders))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	resp, err := json.Marshal(orders)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
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
