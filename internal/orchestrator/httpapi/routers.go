package httpapi

import "net/http"

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

// NewRouter создаёт HTTP-роутер для Orchestrator сервиса
func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", h.Health)

	// POST /sagas — создать новую сагу
	// GET /sagas?id=<uuid> — получить сагу по ID
	mux.HandleFunc("/sagas", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.CreateSaga(w, r)
		case http.MethodGet:
			h.GetSaga(w, r)
		default:
			writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})

	// GET /sagas/list?user_id=<string>
	mux.HandleFunc("/sagas/list", h.ListSagas)

	return cors(mux)
}
