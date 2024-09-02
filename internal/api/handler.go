package api

import "net/http"

type Handler struct{}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	//TODO
	w.WriteHeader(http.StatusOK)
}
