package api

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/felixge/httpsnoop"
	"github.com/google/uuid"
)

func loggingMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := uuid.New().String()
		slog.Info(
			"REQ",
			slog.String("request_id", reqID),
			slog.String("method", r.Method),
			slog.String("uri", r.URL.String()),
			slog.String("ip", r.RemoteAddr),
			slog.String("user_agent", r.Header.Get("User-Agent")),
		)

		// this runs handler h and captures information about HTTP request
		m := httpsnoop.CaptureMetrics(h, w, r)
		slog.Info(
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
