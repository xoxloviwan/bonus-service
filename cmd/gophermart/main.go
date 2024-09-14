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

func mainWithError(cfg conf.Config) error {
	st, err := store.NewStore(context.Background(), cfg.DatabaseURI)
	if err != nil {
		err = fmt.Errorf("unable to create connection pool: %w", err)
		return err
	}
	defer st.Close()
	err = st.CreateUsersTable(context.Background())
	if err != nil {
		return err
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
				return err
			}
			api.Log.Info("service gracefully stopped")
			return nil
		case err := <-errCh:
			return err
		}
	}
}

func main() {
	cfg := conf.InitConfig()
	if err := mainWithError(cfg); err != nil {
		api.Log.Error(fmt.Sprintf("service stopped with error: %s\n", err))
		os.Exit(1)
	}
}
