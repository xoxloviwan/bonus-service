package api

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/felixge/httpsnoop"
)

var Log *slog.Logger
var lvl *slog.LevelVar

var reqID = 0

func init() {
	lvl = new(slog.LevelVar)
	lvl.Set(slog.LevelDebug)
	Log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}

func logger(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID++
		Log.Info(
			"REQ",
			slog.Int("id", reqID),
			slog.String("method", r.Method),
			slog.String("uri", r.URL.String()),
			slog.String("ip", r.RemoteAddr),
			slog.String("user_agent", r.Header.Get("User-Agent")),
		)

		// this runs handler h and captures information about HTTP request
		m := httpsnoop.CaptureMetrics(h, w, r)
		Log.Info(
			"RES",
			slog.Int("id", reqID),
			slog.Int("status", m.Code),
			slog.Duration("duration", m.Duration),
			slog.Int64("size", m.Written),
		)
	})
}
