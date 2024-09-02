package main

import (
	"gophermart/internal/api"
	"log"
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
	log.Printf("service listening on %s", port)

	sigCh := make(chan os.Signal, 1) // we need to reserve to buffer size 1, so the notifier are not blocked
	signal.Notify(sigCh, os.Interrupt, syscall.SIGINT)

	for {
		select {
		case <-sigCh:
			log.Printf("service stopped by SIGINT")
			return
		case err := <-errCh:
			log.Fatalf("service stopped with error:\n%s", err)
		}
	}
}
