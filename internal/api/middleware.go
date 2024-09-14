package api

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/felixge/httpsnoop"
	"github.com/google/uuid"
)

var Log *slog.Logger
var lvl *slog.LevelVar

func init() {
	lvl = new(slog.LevelVar)
	lvl.Set(slog.LevelDebug)
	Log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}

func loggingMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := uuid.New().String()
		Log.Info(
			"REQ",
			slog.String("request_id", reqID),
			slog.String("method", r.Method),
			slog.String("uri", r.URL.String()),
			slog.String("ip", r.RemoteAddr),
			slog.String("user_agent", r.Header.Get("User-Agent")),
		)

		// this runs handler h and captures information about HTTP request
		m := httpsnoop.CaptureMetrics(h, w, r)
		Log.Info(
			"RES",
			slog.String("request_id", reqID),
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
		userID, err := GetUserID(auth)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userIDCtxKey{}, userID)
		r = r.WithContext(ctx)
		h.ServeHTTP(w, r)
	})
}
