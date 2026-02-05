package httpapi

import (
	"net/http"
	"strings"
)

// cors wraps handler with CORS headers
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", h.Health)

	// POST /media (создание)
	mux.HandleFunc("/media", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.CreateMedia(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})

	// GET /media/{id} и PATCH /media/{id}/status
	mux.HandleFunc("/media/", func(w http.ResponseWriter, r *http.Request) {
		// PATCH /media/{id}/status
		if r.Method == http.MethodPatch && strings.HasSuffix(r.URL.Path, "/status") {
			h.ChangeStatus(w, r)
			return
		}

		// GET /media/{id}
		if r.Method == http.MethodGet {
			h.GetMedia(w, r)
			return
		}

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})

	return cors(mux)
}
