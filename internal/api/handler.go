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
type OrderStatus = model.OrderStatus
type Balance = model.Balance
type Payment = model.Payment
type PaymentFact = model.PaymentFact

//go:generate mockgen -destination ../mock/store_mock.go -package mock gophermart/internal/api Store
type Store interface {
	AddUser(ctx context.Context, u User) (int, error)
	GetUser(ctx context.Context, login string) (*User, error)
	AddOrder(ctx context.Context, orderID int, userID int) (OrderStatus, error)
	ListOrders(ctx context.Context, userID int) ([]Order, error)
	GetBalance(ctx context.Context, userID int) (*Balance, error)
	SpendBonus(ctx context.Context, userID int, payment Payment) error
	SpentBonusList(ctx context.Context, userID int) ([]PaymentFact, error)
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
	var status OrderStatus
	status, err = h.store.AddOrder(r.Context(), orderID, userID)
	if err != nil {
		if errors.Is(err, model.ErrOldOrder) {
			if status != model.OrderStatusProcessed && status != model.OrderStatusInvalid {
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

func (h *Handler) Balance(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(userIDCtxKey{}).(int)
	account, err := h.store.GetBalance(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	resp, err := json.Marshal(&account)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(resp)
}

func (h *Handler) Pay(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	slog.Debug(string(body))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var payment Payment
	err = json.Unmarshal(body, &payment)
	slog.Debug(fmt.Sprintf("Payment: %+v", payment))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if payment.OrderID == 0 {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	if payment.Sum == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !luhn.Valid(payment.OrderID) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	userID := r.Context().Value(userIDCtxKey{}).(int)
	err = h.store.SpendBonus(r.Context(), userID, payment)
	if err != nil {
		if errors.Is(err, model.ErrNotEnough) {
			http.Error(w, err.Error(), http.StatusPaymentRequired)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) PaymentList(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(userIDCtxKey{}).(int)
	payments, err := h.store.SpentBonusList(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(payments) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	resp, err := json.Marshal(payments)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}
