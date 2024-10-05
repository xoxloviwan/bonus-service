package main

import (
	"context"
	"fmt"
	"gophermart/internal/api"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	conf "gophermart/internal/config"
	"gophermart/internal/polling"
	"gophermart/internal/store"
)

var lvl *slog.LevelVar

func init() {
	lvl = new(slog.LevelVar)
	lvl.Set(slog.LevelInfo)
	Log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
	slog.SetDefault(Log)
}

func setLogLevel(level string) {
	var lvlVal slog.Level
	switch level {
	case "debug":
		lvlVal = slog.LevelDebug
	case "info":
		lvlVal = slog.LevelInfo
	case "error":
		lvlVal = slog.LevelError
	default:
		lvlVal = slog.LevelDebug
	}
	slog.Info(fmt.Sprintf("log level %s", lvlVal))
	lvl.Set(lvlVal)
}

func mainWithError(cfg conf.Config) error {
	st, err := store.NewStore(context.Background(), cfg.DatabaseURI)
	if err != nil {
		err = fmt.Errorf("unable to create connection pool: %w", err)
		return err
	}
	defer st.Close()
	slog.Info("accrual system info", slog.String("accrual_url", cfg.AccrualSystemAddress))
	pollster := polling.NewPollster(cfg.AccrualSystemAddress, st)

	go pollster.Run(context.Background(), time.Duration(cfg.PollInterval)*time.Second)
	defer pollster.Stop()

	handler := api.NewHandler(st, pollster)
	router := api.Router(handler)
	server := &http.Server{
		Addr:    cfg.RunAddress,
		Handler: router,
	}
	errCh := make(chan error)

	go func() {
		errCh <- server.ListenAndServe()
	}()
	slog.Info(fmt.Sprintf("service listening on %s", cfg.RunAddress))

	sigCh := make(chan os.Signal, 1) // we need to reserve to buffer size 1, so the notifier are not blocked
	signal.Notify(sigCh, os.Interrupt, syscall.SIGINT)

	for {
		select {
		case <-sigCh:
			slog.Info("calling SIGINT")
			ctx, cancelCtx := context.WithTimeout(context.TODO(), 5*time.Second)
			defer cancelCtx()
			err := server.Shutdown(ctx)
			if err != nil {
				slog.Error(fmt.Sprintf("shutdown error: %s", err))
				return err
			}
			slog.Info("service gracefully stopped")
			return nil
		case err := <-errCh:
			return err
		}
	}
}

func main() {
	cfg, err := conf.InitConfig()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	setLogLevel(cfg.Level)
	if err := mainWithError(cfg); err != nil {
		slog.Error(fmt.Sprintf("service stopped with error: %s\n", err))
		os.Exit(1)
	}
}
