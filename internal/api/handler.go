package api

import (
	"encoding/json"
	"net/http"
)

type Store interface {
	AddUser(login string, hash []byte) (int, error)
	GetUser(login string) ([]byte, int, error)
}

type Handler struct {
	store Store
}

type BalanceResponse struct {
	Balance float64 `json:"current"`
	Bonuses int     `json:"withdrawn"`
}

func (h *Handler) Balance(w http.ResponseWriter, r *http.Request) {
	fakeBalance := BalanceResponse{ //TODO
		Balance: 1000,
		Bonuses: 0,
	}
	w.Header().Set("Content-Type", "application/json")
	resp, err := json.Marshal(fakeBalance)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(resp)
}
