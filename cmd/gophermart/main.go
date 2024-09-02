package main

import (
	"fmt"
	"gophermart/internal/api"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

const port = ":8080"

func main() {
	router := api.Router()
	errCh := make(chan error)

	go func() {
		errCh <- http.ListenAndServe(port, router)
	}()
	api.Log.Info(fmt.Sprintf("service listening on %s", port))

	sigCh := make(chan os.Signal, 1) // we need to reserve to buffer size 1, so the notifier are not blocked
	signal.Notify(sigCh, os.Interrupt, syscall.SIGINT)

	for {
		select {
		case <-sigCh:
			api.Log.Info("service stopped by SIGINT")
			return
		case err := <-errCh:
			api.Log.Error(fmt.Sprintf("service stopped with error:\n%s", err))
			os.Exit(1)
		}
	}
}
