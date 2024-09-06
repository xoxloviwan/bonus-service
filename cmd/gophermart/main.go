package main

import (
	"context"
	"fmt"
	"gophermart/internal/api"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gophermart/internal/store"
)

const PORT = ":8080"
const DB_URL = "postgresql://postgres:12345@localhost:5432/postgres?sslmode=disable"

func fatal(err error) int {
	api.Log.Error(fmt.Sprintf("service stopped with error: %s\n", err))
	return 1
}

func mainWithExitCode() int {
	st, err := store.NewStore(context.Background(), DB_URL)
	if err != nil {
		err = fmt.Errorf("unable to create connection pool: %w", err)
		return fatal(err)
	}
	defer st.Close()
	err = st.CreateUsersTable(context.Background())
	if err != nil {
		return fatal(err)
	}

	router := api.Router(st)
	server := &http.Server{
		Addr:    PORT,
		Handler: router,
	}
	errCh := make(chan error)

	go func() {
		errCh <- server.ListenAndServe()
	}()
	api.Log.Info(fmt.Sprintf("service listening on %s", PORT))

	sigCh := make(chan os.Signal, 1) // we need to reserve to buffer size 1, so the notifier are not blocked
	signal.Notify(sigCh, os.Interrupt, syscall.SIGINT)

	for {
		select {
		case <-sigCh:
			api.Log.Info("calling SIGINT")
			ctx, cancelCtx := context.WithTimeout(context.TODO(), 5*time.Second)
			defer cancelCtx()
			err := server.Shutdown(ctx)
			if err != nil {
				api.Log.Error(fmt.Sprintf("shutdown error: %s", err))
				return 1
			}
			api.Log.Info("service gracefully stopped")
			return 0
		case err := <-errCh:
			return fatal(err)
		}
	}
}

func main() {
	code := mainWithExitCode()
	os.Exit(code)
}
