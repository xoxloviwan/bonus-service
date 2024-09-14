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

	conf "gophermart/internal/config"
	"gophermart/internal/store"
)

func fatal(err error) int {
	api.Log.Error(fmt.Sprintf("service stopped with error: %s\n", err))
	return 1
}

func mainWithExitCode(cfg conf.Config) int {
	st, err := store.NewStore(context.Background(), cfg.DatabaseURI)
	if err != nil {
		err = fmt.Errorf("unable to create connection pool: %w", err)
		return fatal(err)
	}
	defer st.Close()
	err = st.CreateUsersTable(context.Background())
	if err != nil {
		return fatal(err)
	}

	handler := api.NewHandler(st)
	router := api.Router(handler)
	server := &http.Server{
		Addr:    cfg.RunAddress,
		Handler: router,
	}
	errCh := make(chan error)

	go func() {
		errCh <- server.ListenAndServe()
	}()
	api.Log.Info(fmt.Sprintf("service listening on %s", cfg.RunAddress))

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
	cfg := conf.InitConfig()
	if err := mainWithExitCode(cfg); err != nil {
	    os.Exit(1)
	}
}
