package api

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	*pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) Store {
	return Store{pool}
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
