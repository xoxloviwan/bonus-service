package api

import (
	"net/http"

	"github.com/go-pkgz/routegroup"
)

func Router(h *Handler) http.Handler {
	router := routegroup.New(http.NewServeMux())
	router.Use(loggingMiddleware)

	// create a new group for the /api/user path
	apiRouter := router.Mount("/api/user")
	apiRouter.HandleFunc("POST /register", h.Register)
	apiRouter.HandleFunc("POST /login", h.Login)

	protectedGroup := apiRouter.Group()
	protectedGroup.Use(authMiddleware)
	protectedGroup.HandleFunc("POST /orders", h.NewOrder)
	protectedGroup.HandleFunc("GET /orders", h.OrderList)
	protectedGroup.HandleFunc("GET /balance", h.Balance)
	protectedGroup.HandleFunc("POST /balance/withdraw", h.Pay)

	return router
}
