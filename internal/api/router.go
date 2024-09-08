package api

import (
	"net/http"

	"github.com/go-pkgz/routegroup"
)

func Router(store Store) http.Handler {
	h := Handler{store: store}
	router := routegroup.New(http.NewServeMux())
	router.Use(loggingMiddleware)

	// create a new group for the /api/user path
	apiRouter := router.Mount("/api/user")
	apiRouter.HandleFunc("POST /register", h.Register)
	apiRouter.HandleFunc("POST /login", h.Login)

	protectedGroup := apiRouter.Group()
	protectedGroup.Use(authMiddleware)
	protectedGroup.HandleFunc("POST /order", h.NewOrder)
	protectedGroup.HandleFunc("GET /balance", h.Balance)

	return router
}
