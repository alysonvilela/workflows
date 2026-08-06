// Package httpapi wires the chi router and HTTP handlers.
package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Pinger is satisfied by both store.Store and storage.Storage.
type Pinger interface {
	Ping(ctx context.Context) error
}

func NewRouter(db Pinger, objectStore Pinger, corsAllowedOrigins string) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors(corsAllowedOrigins))

	r.Get("/health", healthHandler(db, objectStore))

	return r
}

func cors(allowedOrigins string) func(http.Handler) http.Handler {
	origins := strings.Split(allowedOrigins, ",")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			origin := req.Header.Get("Origin")
			for _, allowed := range origins {
				if strings.TrimSpace(allowed) == origin {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
					break
				}
			}
			if req.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, req)
		})
	}
}

type healthStatus struct {
	Status   string `json:"status"`
	Database string `json:"database"`
	Storage  string `json:"storage"`
}

func healthHandler(db Pinger, objectStore Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body := healthStatus{Status: "ok", Database: "ok", Storage: "ok"}
		code := http.StatusOK

		if err := db.Ping(r.Context()); err != nil {
			body.Database = err.Error()
			body.Status = "degraded"
			code = http.StatusServiceUnavailable
		}
		if err := objectStore.Ping(r.Context()); err != nil {
			body.Storage = err.Error()
			body.Status = "degraded"
			code = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(body)
	}
}
