package server

import (
	"database/sql"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"

	"study-help/internal/config"
	"study-help/internal/esv"
	webclient "study-help/internal/web"
)

// New builds the public HTTP server: health, the passage proxy, and the
// static SPA bundle. The /metrics endpoint runs on a separate localhost
// listener — see NewMetricsServer.
func New(cfg config.Config, db *sql.DB, counter *ESVCallCounter) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := db.PingContext(r.Context()); err != nil {
			http.Error(w, "db unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	esvClient := esv.NewClient(cfg.ESVAPIKey)
	mux.HandleFunc("GET /api/passage", passageHandler(esvClient, counter))

	mux.Handle("/", spaHandler(webclient.DistFS()))

	return &http.Server{
		Addr:              cfg.Addr,
		Handler:           logging(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
}

// spaHandler serves the Vite static build. Unknown non-asset paths fall
// back to index.html so client-side routing works. Reserved prefixes
// (/api/, /healthz, /metrics) get a 404 instead of the SPA shell so an
// unknown API path is obviously broken rather than silently rendering
// the app.
func spaHandler(dist fs.FS) http.Handler {
	if dist == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "client bundle not built — run `npm run build` in web/", http.StatusNotFound)
		})
	}
	fileServer := http.FileServer(http.FS(dist))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") ||
			r.URL.Path == "/healthz" ||
			r.URL.Path == "/metrics" {
			http.NotFound(w, r)
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			fileServer.ServeHTTP(w, r)
			return
		}
		if _, err := fs.Stat(dist, path); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/"
		fileServer.ServeHTTP(w, r2)
	})
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, rec.status, time.Since(start))
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
