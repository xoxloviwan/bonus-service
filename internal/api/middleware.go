package api

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"sync/atomic"

	"github.com/felixge/httpsnoop"
)

var Log *slog.Logger
var lvl *slog.LevelVar

var reqNum atomic.Uint64

func init() {
	lvl = new(slog.LevelVar)
	lvl.Set(slog.LevelDebug)
	Log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}

func loggingMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := reqNum.Load()
		reqNum.Add(1)
		Log.Info(
			"REQ",
			slog.Uint64("id", reqID),
			slog.String("method", r.Method),
			slog.String("uri", r.URL.String()),
			slog.String("ip", r.RemoteAddr),
			slog.String("user_agent", r.Header.Get("User-Agent")),
		)

		// this runs handler h and captures information about HTTP request
		m := httpsnoop.CaptureMetrics(h, w, r)
		Log.Info(
			"RES",
			slog.Uint64("id", reqID),
			slog.Int("status", m.Code),
			slog.Duration("duration", m.Duration),
			slog.Int64("size", m.Written),
		)
	})
}

type userIDCtxKey struct{}

func authMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		auth = auth[7:] // cut 'Bearer ' prefix
		userID, err := GetUserId(auth)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userIDCtxKey{}, userID)
		r = r.WithContext(ctx)
		h.ServeHTTP(w, r)
	})
}
