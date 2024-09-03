package api

import (
	"net/http"

	"github.com/go-pkgz/routegroup"
)

func Router() http.Handler {
	h := Handler{}
	router := routegroup.New(http.NewServeMux())
	router.Use(loggingMiddleware)

	// create a new group for the /api/user path
	apiRouter := router.Mount("/api/user")
	apiRouter.HandleFunc("POST /register", h.Register)
	apiRouter.HandleFunc("POST /login", h.Login)

	protectedGroup := apiRouter.Group()
	protectedGroup.Use(authMiddleware)
	protectedGroup.HandleFunc("GET /balance", h.Balance)

	return router
}
